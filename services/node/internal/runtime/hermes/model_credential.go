package hermes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	modelCredentialFileName = ".env"
	modelCredentialMaxFile  = 256 * 1024
	modelCredentialMaxValue = 16 * 1024
)

var (
	errModelCredentialInvalid  = errors.New("hermes model credential is invalid")
	errModelCredentialUnsafe   = errors.New("hermes model credential store is unsafe")
	errModelCredentialConflict = errors.New("hermes model credential changed concurrently")
	errModelCredentialWrite    = errors.New("hermes model credential write failed")
	errModelCredentialQuery    = errors.New("hermes model credential query failed")
)

type ModelCredentialStatus struct {
	ProviderPresetID string
	Configured       bool
}

func (m *ModelManager) ModelCredentialStatus(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID, presetID string) (yorvaruntime.ModelCredentialStatus, error) {
	if err := validateModelCredentialTarget(ctx, installation, nativeID); err != nil {
		return yorvaruntime.ModelCredentialStatus{}, err
	}
	status, err := m.credentials.Status(nativeID, presetID)
	if err != nil {
		return yorvaruntime.ModelCredentialStatus{}, normalizeModelCredentialError(err, false)
	}
	return yorvaruntime.ModelCredentialStatus{ProviderPresetID: status.ProviderPresetID, Configured: status.Configured}, nil
}

func (m *ModelManager) SetModelCredential(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID, presetID string, secret []byte) (yorvaruntime.ModelCredentialStatus, error) {
	if err := validateModelCredentialTarget(ctx, installation, nativeID); err != nil {
		return yorvaruntime.ModelCredentialStatus{}, err
	}
	status, err := m.credentials.Set(nativeID, presetID, secret)
	if err != nil {
		return yorvaruntime.ModelCredentialStatus{}, normalizeModelCredentialError(err, false)
	}
	return yorvaruntime.ModelCredentialStatus{ProviderPresetID: status.ProviderPresetID, Configured: status.Configured}, nil
}

func (m *ModelManager) DeleteModelCredential(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID, presetID string) (yorvaruntime.ModelCredentialStatus, error) {
	if err := validateModelCredentialTarget(ctx, installation, nativeID); err != nil {
		return yorvaruntime.ModelCredentialStatus{}, err
	}
	status, err := m.credentials.Delete(nativeID, presetID)
	if err != nil {
		return yorvaruntime.ModelCredentialStatus{}, normalizeModelCredentialError(err, true)
	}
	return yorvaruntime.ModelCredentialStatus{ProviderPresetID: status.ProviderPresetID, Configured: status.Configured}, nil
}

func validateModelCredentialTarget(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return validateModelTarget(installation, nativeID)
}

func normalizeModelCredentialError(err error, deleting bool) error {
	switch {
	case IsModelProviderUnsupported(err):
		return yorvaruntime.ErrModelProviderUnsupported
	case IsModelCredentialInvalid(err):
		return yorvaruntime.ErrModelConfigInvalid
	case IsModelCredentialConflict(err):
		return yorvaruntime.ErrInstanceConfigConflict
	case IsModelCredentialQueryFailure(err), IsModelCredentialUnsafe(err):
		return yorvaruntime.ErrModelCredentialQueryFailed
	case deleting:
		return yorvaruntime.ErrModelCredentialDeleteFailed
	default:
		return yorvaruntime.ErrModelCredentialWriteFailed
	}
}

type credentialSnapshot struct {
	exists bool
	data   []byte
	mode   os.FileMode
}

// credentialStore is deliberately adapter-private. Callers provide only a
// native Profile id and allowlisted preset id; no path or environment name can
// cross the Hermes boundary.
type credentialStore struct {
	root          string
	beforeReplace func()
}

func newModelCredentialStore() credentialStore {
	return credentialStore{root: officialHermesHome()}
}

func ModelCredentialStatusFor(version, nativeID, presetID string) (ModelCredentialStatus, error) {
	if version != modelSurfaceVersion {
		return ModelCredentialStatus{}, errModelVersionUnsupported
	}
	return newModelCredentialStore().Status(nativeID, presetID)
}

func SetModelCredential(version, nativeID, presetID string, secret []byte) (ModelCredentialStatus, error) {
	if version != modelSurfaceVersion {
		return ModelCredentialStatus{}, errModelVersionUnsupported
	}
	return newModelCredentialStore().Set(nativeID, presetID, secret)
}

func DeleteModelCredential(version, nativeID, presetID string) (ModelCredentialStatus, error) {
	if version != modelSurfaceVersion {
		return ModelCredentialStatus{}, errModelVersionUnsupported
	}
	return newModelCredentialStore().Delete(nativeID, presetID)
}

func (s credentialStore) Status(nativeID, presetID string) (ModelCredentialStatus, error) {
	preset, err := lookupModelProviderPreset(presetID)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	path, err := s.credentialPath(nativeID)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	configured, err := credentialConfigured(snapshot.data, preset.credentialEnvName)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	return ModelCredentialStatus{ProviderPresetID: presetID, Configured: configured}, nil
}

func (s credentialStore) Set(nativeID, presetID string, secret []byte) (ModelCredentialStatus, error) {
	preset, err := lookupModelProviderPreset(presetID)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	if err := validateModelCredential(secret); err != nil {
		return ModelCredentialStatus{}, err
	}
	path, err := s.credentialPath(nativeID)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	updated, err := replaceCredentialAssignment(snapshot.data, preset.credentialEnvName, secret)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	defer clearCredentialBytes(updated)
	if bytes.Equal(updated, snapshot.data) {
		return ModelCredentialStatus{ProviderPresetID: presetID, Configured: true}, nil
	}
	if err := s.commit(path, snapshot, updated); err != nil {
		return ModelCredentialStatus{}, err
	}
	return ModelCredentialStatus{ProviderPresetID: presetID, Configured: true}, nil
}

func (s credentialStore) Delete(nativeID, presetID string) (ModelCredentialStatus, error) {
	preset, err := lookupModelProviderPreset(presetID)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	path, err := s.credentialPath(nativeID)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	updated, found, err := deleteCredentialAssignment(snapshot.data, preset.credentialEnvName)
	if err != nil {
		return ModelCredentialStatus{}, err
	}
	if !found {
		return ModelCredentialStatus{ProviderPresetID: presetID}, nil
	}
	if err := s.commit(path, snapshot, updated); err != nil {
		return ModelCredentialStatus{}, err
	}
	return ModelCredentialStatus{ProviderPresetID: presetID}, nil
}

func (s credentialStore) credentialPath(nativeID string) (string, error) {
	if s.root == "" || !filepath.IsAbs(s.root) {
		return "", errModelCredentialUnsafe
	}
	normalized, err := officialNormalizeProfileName(nativeID)
	if err != nil || officialValidateProfileName(normalized) != nil {
		return "", errModelCredentialUnsafe
	}
	profileRoot := s.root
	if normalized != "default" {
		profileRoot = filepath.Join(s.root, "profiles", normalized)
	}
	if err := rejectReparsePoint(profileRoot); err != nil {
		return "", errModelCredentialUnsafe
	}
	info, err := os.Lstat(profileRoot)
	if err != nil || !info.IsDir() {
		return "", errModelCredentialUnsafe
	}
	path := filepath.Join(profileRoot, modelCredentialFileName)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errModelCredentialUnsafe
	}
	return path, nil
}

func observeCredentialFile(path string) (credentialSnapshot, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return credentialSnapshot{}, nil
	}
	if err != nil {
		return credentialSnapshot{}, errModelCredentialQuery
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) || info.Size() > modelCredentialMaxFile {
		return credentialSnapshot{}, errModelCredentialUnsafe
	}
	file, err := os.Open(path)
	if err != nil {
		return credentialSnapshot{}, errModelCredentialQuery
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, modelCredentialMaxFile+1))
	if err != nil {
		return credentialSnapshot{}, errModelCredentialQuery
	}
	if len(data) > modelCredentialMaxFile || !utf8.Valid(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})) {
		return credentialSnapshot{}, errModelCredentialUnsafe
	}
	return credentialSnapshot{exists: true, data: data, mode: info.Mode().Perm()}, nil
}

func (s credentialStore) commit(path string, expected credentialSnapshot, payload []byte) (err error) {
	current, err := observeCredentialFile(path)
	if err != nil {
		return err
	}
	if !snapshotsEqual(current, expected) {
		return errModelCredentialConflict
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".yorva-model-credential-*")
	if err != nil {
		return errModelCredentialWrite
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()
	if err := temp.Chmod(0o600); err != nil {
		return errModelCredentialWrite
	}
	if _, err := temp.Write(payload); err != nil {
		return errModelCredentialWrite
	}
	if err := temp.Sync(); err != nil {
		return errModelCredentialWrite
	}
	if err := temp.Close(); err != nil {
		return errModelCredentialWrite
	}
	if s.beforeReplace != nil {
		s.beforeReplace()
	}
	current, err = observeCredentialFile(path)
	if err != nil {
		return err
	}
	if !snapshotsEqual(current, expected) {
		return errModelCredentialConflict
	}
	if err := atomicReplaceCredentialFile(tempPath, path); err != nil {
		return errModelCredentialWrite
	}
	observed, err := observeCredentialFile(path)
	if err != nil || !observed.exists || !bytes.Equal(observed.data, payload) {
		return errModelCredentialWrite
	}
	return nil
}

func snapshotsEqual(left, right credentialSnapshot) bool {
	return left.exists == right.exists && left.mode == right.mode && bytes.Equal(left.data, right.data)
}

func validateModelCredential(secret []byte) error {
	if len(secret) < 4 || len(secret) > modelCredentialMaxValue || !utf8.Valid(secret) || bytes.Contains(secret, []byte("${")) {
		return errModelCredentialInvalid
	}
	value := string(secret)
	if strings.TrimSpace(value) != value || isCredentialPlaceholder(value) {
		return errModelCredentialInvalid
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return errModelCredentialInvalid
		}
	}
	return nil
}

func credentialConfigured(data []byte, name string) (bool, error) {
	lines := splitCredentialLines(data)
	configured := false
	found := false
	for _, line := range lines {
		value, match := credentialAssignment(line.body, name)
		if !match {
			continue
		}
		if found {
			return false, errModelCredentialUnsafe
		}
		found = true
		parsed, parseErr := assignmentHasValue(value)
		if parseErr != nil {
			return false, parseErr
		}
		configured = parsed
	}
	return configured, nil
}

func replaceCredentialAssignment(data []byte, name string, secret []byte) ([]byte, error) {
	content, bom := stripCredentialBOM(data)
	lines := splitCredentialLines(content)
	found := -1
	for index, line := range lines {
		if _, match := credentialAssignment(line.body, name); match {
			if found >= 0 {
				return nil, errModelCredentialUnsafe
			}
			found = index
		}
	}
	assignment := []byte(name + "=" + quoteCredentialValue(secret))
	if found >= 0 {
		lines[found].body = assignment
		return restoreCredentialBOM(joinCredentialLines(lines), bom), nil
	}
	result := append([]byte(nil), content...)
	if len(result) > 0 && result[len(result)-1] != '\n' && result[len(result)-1] != '\r' {
		result = append(result, '\n')
	}
	result = append(result, assignment...)
	result = append(result, '\n')
	return restoreCredentialBOM(result, bom), nil
}

func deleteCredentialAssignment(data []byte, name string) ([]byte, bool, error) {
	content, bom := stripCredentialBOM(data)
	lines := splitCredentialLines(content)
	found := -1
	for index, line := range lines {
		if _, match := credentialAssignment(line.body, name); match {
			if found >= 0 {
				return nil, false, errModelCredentialUnsafe
			}
			found = index
		}
	}
	if found < 0 {
		return append([]byte(nil), data...), false, nil
	}
	lines = append(lines[:found], lines[found+1:]...)
	return restoreCredentialBOM(joinCredentialLines(lines), bom), true, nil
}

func stripCredentialBOM(data []byte) ([]byte, bool) {
	bom := []byte{0xef, 0xbb, 0xbf}
	if bytes.HasPrefix(data, bom) {
		return data[len(bom):], true
	}
	return data, false
}

func restoreCredentialBOM(data []byte, present bool) []byte {
	if !present {
		return data
	}
	return append([]byte{0xef, 0xbb, 0xbf}, data...)
}

type credentialLine struct {
	body, ending []byte
}

func splitCredentialLines(data []byte) []credentialLine {
	var lines []credentialLine
	for len(data) > 0 {
		index := bytes.IndexByte(data, '\n')
		if index < 0 {
			lines = append(lines, credentialLine{body: append([]byte(nil), data...)})
			break
		}
		bodyEnd := index
		ending := []byte{'\n'}
		if bodyEnd > 0 && data[bodyEnd-1] == '\r' {
			bodyEnd--
			ending = []byte{'\r', '\n'}
		}
		lines = append(lines, credentialLine{
			body: append([]byte(nil), data[:bodyEnd]...), ending: append([]byte(nil), ending...),
		})
		data = data[index+1:]
	}
	return lines
}

func joinCredentialLines(lines []credentialLine) []byte {
	var result []byte
	for _, line := range lines {
		result = append(result, line.body...)
		result = append(result, line.ending...)
	}
	return result
}

func credentialAssignment(line []byte, name string) ([]byte, bool) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(line, []byte{0xef, 0xbb, 0xbf}))
	if bytes.HasPrefix(trimmed, []byte("export ")) {
		trimmed = bytes.TrimSpace(trimmed[len("export "):])
	}
	key, value, ok := bytes.Cut(trimmed, []byte{'='})
	if !ok || string(bytes.TrimSpace(key)) != name {
		return nil, false
	}
	return bytes.TrimSpace(value), true
}

func assignmentHasValue(value []byte) (bool, error) {
	value = bytes.TrimSpace(value)
	if len(value) == 0 {
		return false, nil
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		closing := -1
		escaped := false
		for index := 1; index < len(value); index++ {
			if quote == '"' && value[index] == '\\' && !escaped {
				escaped = true
				continue
			}
			if value[index] == quote && !escaped {
				closing = index
				break
			}
			escaped = false
		}
		if closing < 0 {
			return false, errModelCredentialUnsafe
		}
		tail := bytes.TrimSpace(value[closing+1:])
		if len(tail) > 0 && tail[0] != '#' {
			return false, errModelCredentialUnsafe
		}
		value = value[1:closing]
	} else {
		for index := 0; index < len(value); index++ {
			if value[index] == '#' && (index == 0 || value[index-1] == ' ' || value[index-1] == '\t') {
				value = bytes.TrimSpace(value[:index])
				break
			}
		}
	}
	if bytes.Contains(value, []byte("${")) {
		return false, errModelCredentialUnsafe
	}
	cleaned := strings.TrimSpace(string(value))
	return len(cleaned) >= 4 && !isCredentialPlaceholder(cleaned), nil
}

func clearCredentialBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

func isCredentialPlaceholder(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "*", "**", "***", "changeme", "your_api_key", "your_api_key_here", "your-api-key", "placeholder", "example", "dummy", "null", "none":
		return true
	default:
		return false
	}
}

func quoteCredentialValue(secret []byte) string {
	value := strings.ReplaceAll(string(secret), `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return fmt.Sprintf(`"%s"`, value)
}

func IsModelCredentialInvalid(err error) bool      { return errors.Is(err, errModelCredentialInvalid) }
func IsModelCredentialUnsafe(err error) bool       { return errors.Is(err, errModelCredentialUnsafe) }
func IsModelCredentialConflict(err error) bool     { return errors.Is(err, errModelCredentialConflict) }
func IsModelCredentialWriteFailure(err error) bool { return errors.Is(err, errModelCredentialWrite) }
func IsModelCredentialQueryFailure(err error) bool { return errors.Is(err, errModelCredentialQuery) }
