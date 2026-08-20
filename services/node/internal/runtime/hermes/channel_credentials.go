package hermes

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const channelCredentialVersion = "0.20.2"

var channelIdentityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:@-]{0,127}$`)

type channelCredentialStore struct {
	credentials credentialStore
}

type weixinAccount struct {
	Token   string `json:"token"`
	BaseURL string `json:"base_url"`
	UserID  string `json:"user_id"`
	SavedAt string `json:"saved_at"`
}

func newChannelCredentialStore() channelCredentialStore {
	return channelCredentialStore{credentials: newModelCredentialStore()}
}

func (s channelCredentialStore) Status(nativeID string, kind channel.Type) (yorvaruntime.ChannelStatus, error) {
	assignments, err := s.readAssignments(nativeID)
	if err != nil {
		return yorvaruntime.ChannelStatus{}, err
	}
	switch kind {
	case channel.Weixin:
		accountID := assignments["WEIXIN_ACCOUNT_ID"]
		token := assignments["WEIXIN_TOKEN"]
		if accountID == "" && token == "" {
			return yorvaruntime.ChannelStatus{Type: kind, State: channel.NotConfigured}, nil
		}
		if !channelIdentityPattern.MatchString(accountID) || token == "" {
			return yorvaruntime.ChannelStatus{Type: kind, State: channel.Unknown}, nil
		}
		account, err := s.readWeixinAccount(nativeID, accountID)
		if err != nil || account.Token != token || !allowedWeixinBaseURL(account.BaseURL) {
			return yorvaruntime.ChannelStatus{Type: kind, State: channel.Unknown}, nil
		}
		return yorvaruntime.ChannelStatus{Type: kind, State: channel.Unknown, AccountLabel: safeChannelLabel(account.UserID), ExternalID: accountID}, nil
	case channel.WeCom:
		botID := assignments["WECOM_BOT_ID"]
		secret := assignments["WECOM_SECRET"]
		if botID == "" && secret == "" {
			return yorvaruntime.ChannelStatus{Type: kind, State: channel.NotConfigured}, nil
		}
		if !channelIdentityPattern.MatchString(botID) || secret == "" {
			return yorvaruntime.ChannelStatus{Type: kind, State: channel.Unknown}, nil
		}
		return yorvaruntime.ChannelStatus{Type: kind, State: channel.Unknown, AccountLabel: safeChannelLabel(botID), ExternalID: botID}, nil
	default:
		return yorvaruntime.ChannelStatus{}, errors.New("unsupported channel credential")
	}
}

func (s channelCredentialStore) SetWeixin(nativeID, accountID string, token []byte, baseURL, userID string) error {
	if !channelIdentityPattern.MatchString(accountID) || len(token) == 0 || len(token) > modelCredentialMaxValue || !allowedWeixinBaseURL(baseURL) || (userID != "" && !channelIdentityPattern.MatchString(userID)) {
		return errModelCredentialInvalid
	}
	if err := validateModelCredential(token); err != nil {
		return err
	}
	values := map[string][]byte{
		"WEIXIN_ACCOUNT_ID":      []byte(accountID),
		"WEIXIN_TOKEN":           token,
		"WEIXIN_BASE_URL":        []byte(strings.TrimRight(baseURL, "/")),
		"WEIXIN_CDN_BASE_URL":    []byte("https://novac2c.cdn.weixin.qq.com/c2c"),
		"WEIXIN_DM_POLICY":       []byte("pairing"),
		"WEIXIN_ALLOW_ALL_USERS": []byte("false"),
	}
	if err := s.setAssignments(nativeID, values); err != nil {
		return err
	}
	account := weixinAccount{Token: string(token), BaseURL: strings.TrimRight(baseURL, "/"), UserID: userID, SavedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := s.writeWeixinAccount(nativeID, accountID, account); err != nil {
		_ = s.deleteAssignments(nativeID, keys(values))
		return err
	}
	return nil
}

func (s channelCredentialStore) SetWeCom(nativeID, botID string, secret []byte) error {
	if !channelIdentityPattern.MatchString(botID) || len(secret) == 0 || len(secret) > modelCredentialMaxValue {
		return errModelCredentialInvalid
	}
	if err := validateModelCredential(secret); err != nil {
		return err
	}
	return s.setAssignments(nativeID, map[string][]byte{
		"WECOM_BOT_ID":       []byte(botID),
		"WECOM_SECRET":       secret,
		"WECOM_DM_POLICY":    []byte("pairing"),
		"WECOM_GROUP_POLICY": []byte("pairing"),
	})
}

func (s channelCredentialStore) Delete(nativeID string, kind channel.Type) error {
	switch kind {
	case channel.Weixin:
		assignments, err := s.readAssignments(nativeID)
		if err != nil {
			return err
		}
		accountID := assignments["WEIXIN_ACCOUNT_ID"]
		if err := s.deleteAssignments(nativeID, []string{"WEIXIN_ACCOUNT_ID", "WEIXIN_TOKEN", "WEIXIN_BASE_URL", "WEIXIN_CDN_BASE_URL", "WEIXIN_DM_POLICY", "WEIXIN_ALLOW_ALL_USERS", "WEIXIN_ALLOWED_USERS"}); err != nil {
			return err
		}
		if accountID != "" && channelIdentityPattern.MatchString(accountID) {
			path, pathErr := s.weixinAccountPath(nativeID, accountID)
			if pathErr != nil {
				return pathErr
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return errModelCredentialWrite
			}
		}
		return nil
	case channel.WeCom:
		return s.deleteAssignments(nativeID, []string{"WECOM_BOT_ID", "WECOM_SECRET", "WECOM_WEBSOCKET_URL", "WECOM_DM_POLICY", "WECOM_GROUP_POLICY", "WECOM_ALLOWED_USERS"})
	default:
		return errModelCredentialInvalid
	}
}

func (s channelCredentialStore) readAssignments(nativeID string) (map[string]string, error) {
	path, err := s.credentials.credentialPath(nativeID)
	if err != nil {
		return nil, err
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return nil, err
	}
	return parseSafeEnvAssignments(snapshot.data)
}

func (s channelCredentialStore) setAssignments(nativeID string, values map[string][]byte) error {
	path, err := s.credentials.credentialPath(nativeID)
	if err != nil {
		return err
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return err
	}
	updated := append([]byte(nil), snapshot.data...)
	defer clearCredentialBytes(updated)
	for _, key := range sortedKeys(values) {
		next, replaceErr := replaceCredentialAssignment(updated, key, values[key])
		if replaceErr != nil {
			return replaceErr
		}
		clearCredentialBytes(updated)
		updated = next
	}
	if bytes.Equal(updated, snapshot.data) {
		return nil
	}
	return s.credentials.commit(path, snapshot, updated)
}

func (s channelCredentialStore) deleteAssignments(nativeID string, names []string) error {
	path, err := s.credentials.credentialPath(nativeID)
	if err != nil {
		return err
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return err
	}
	updated := append([]byte(nil), snapshot.data...)
	changed := false
	for _, name := range names {
		next, found, deleteErr := deleteCredentialAssignment(updated, name)
		if deleteErr != nil {
			return deleteErr
		}
		if found {
			clearCredentialBytes(updated)
			updated = next
			changed = true
		}
	}
	defer clearCredentialBytes(updated)
	if !changed {
		return nil
	}
	return s.credentials.commit(path, snapshot, updated)
}

func (s channelCredentialStore) weixinAccountPath(nativeID, accountID string) (string, error) {
	envPath, err := s.credentials.credentialPath(nativeID)
	if err != nil || !channelIdentityPattern.MatchString(accountID) {
		return "", errModelCredentialUnsafe
	}
	root := filepath.Dir(envPath)
	dir := filepath.Join(root, "weixin", "accounts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", errModelCredentialWrite
	}
	if err := rejectReparsePoint(root); err != nil {
		return "", errModelCredentialUnsafe
	}
	if err := rejectReparsePoint(filepath.Join(root, "weixin")); err != nil {
		return "", errModelCredentialUnsafe
	}
	if err := rejectReparsePoint(dir); err != nil {
		return "", errModelCredentialUnsafe
	}
	return filepath.Join(dir, accountID+".json"), nil
}

func (s channelCredentialStore) readWeixinAccount(nativeID, accountID string) (weixinAccount, error) {
	path, err := s.weixinAccountPath(nativeID, accountID)
	if err != nil {
		return weixinAccount{}, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > modelCredentialMaxFile || isReparsePoint(info) {
		return weixinAccount{}, errModelCredentialUnsafe
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return weixinAccount{}, errModelCredentialQuery
	}
	defer clearCredentialBytes(data)
	var account weixinAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return weixinAccount{}, errModelCredentialUnsafe
	}
	return account, nil
}

func (s channelCredentialStore) writeWeixinAccount(nativeID, accountID string, account weixinAccount) error {
	path, err := s.weixinAccountPath(nativeID, accountID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(account)
	if err != nil {
		return errModelCredentialWrite
	}
	defer clearCredentialBytes(payload)
	temp, err := os.CreateTemp(filepath.Dir(path), ".yorva-weixin-account-*")
	if err != nil {
		return errModelCredentialWrite
	}
	tempPath := temp.Name()
	defer func() { _ = temp.Close(); _ = os.Remove(tempPath) }()
	if temp.Chmod(0o600) != nil || writeAndSync(temp, payload) != nil || temp.Close() != nil || atomicReplaceCredentialFile(tempPath, path) != nil {
		return errModelCredentialWrite
	}
	return nil
}

func parseSafeEnvAssignments(data []byte) (map[string]string, error) {
	result := make(map[string]string)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(name) != name {
			continue
		}
		if _, duplicate := result[name]; duplicate {
			return nil, errModelCredentialUnsafe
		}
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			var decoded string
			if json.Unmarshal([]byte(value), &decoded) != nil {
				return nil, errModelCredentialUnsafe
			}
			result[name] = decoded
		} else {
			result[name] = value
		}
	}
	return result, nil
}

func safeChannelLabel(value string) string {
	if len(value) <= 48 {
		return value
	}
	return value[:48]
}

func keys(values map[string][]byte) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}

func sortedKeys(values map[string][]byte) []string {
	out := keys(values)
	slices.Sort(out)
	return out
}

func writeAndSync(file *os.File, payload []byte) error {
	if _, err := file.Write(payload); err != nil {
		return err
	}
	return file.Sync()
}
