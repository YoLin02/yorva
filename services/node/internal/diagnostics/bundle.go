package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/buildinfo"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	SchemaVersion        = "1"
	MaxOperations        = 50
	MaxInstances         = 100
	MaxLogRecords        = 512
	MaxLogLineBytes      = 64 * 1024
	MaxUncompressedBytes = 2 * 1024 * 1024
	MaxBundleBytes       = 2 * 1024 * 1024
	LogWindow            = 7 * 24 * time.Hour
)

var ErrBundleTooLarge = errors.New("diagnostic bundle exceeds its fixed size limit")

type RuntimeReader interface {
	Detect(context.Context, yorvaruntime.Kind) (yorvaruntime.Discovery, error)
}

type InstanceReader interface {
	ListInstances(context.Context, string) (app.InstanceList, error)
}

type OperationReader interface {
	List(context.Context, string, string, int) ([]operation.Operation, error)
}

type SchemaReader interface {
	SchemaVersion(context.Context) (int, error)
}

type Service struct {
	node       node.Node
	runtimes   RuntimeReader
	instances  InstanceReader
	operations OperationReader
	schema     SchemaReader
	dataDir    string
	now        func() time.Time
}

type Bundle struct {
	Bytes     []byte
	FileName  string
	CreatedAt time.Time
}

type redactionReport struct {
	PolicyVersion             string `json:"policyVersion"`
	IdentifiersHashed         int    `json:"identifiersHashed"`
	FieldsDropped             int    `json:"fieldsDropped"`
	ValuesRedacted            int    `json:"valuesRedacted"`
	InstanceRecordsTruncated  int    `json:"instanceRecordsTruncated"`
	OperationRecordsTruncated int    `json:"operationRecordsTruncated"`
	LogRecordsRead            int    `json:"logRecordsRead"`
	LogRecordsWritten         int    `json:"logRecordsWritten"`
	LogRecordsOutsideWindow   int    `json:"logRecordsOutsideWindow"`
	LogRecordsTruncated       int    `json:"logRecordsTruncated"`
}

func New(node node.Node, runtimes RuntimeReader, instances InstanceReader, operations OperationReader, schema SchemaReader, dataDir string) *Service {
	return &Service{node: node, runtimes: runtimes, instances: instances, operations: operations, schema: schema, dataDir: dataDir, now: time.Now}
}

func (s *Service) Build(ctx context.Context) (Bundle, error) {
	now := s.now().UTC()
	report := &redactionReport{PolicyVersion: SchemaVersion}
	runtimeSummary := map[string]any{"runtimeKind": "hermes", "state": "UNKNOWN", "errorCode": "DIAGNOSTICS_RUNTIME_READ_FAILED"}
	runtimeCtx, cancelRuntime := context.WithTimeout(ctx, 10*time.Second)
	discovery, runtimeErr := s.runtimes.Detect(runtimeCtx, "hermes")
	cancelRuntime()
	if runtimeErr == nil {
		runtimeSummary = map[string]any{
			"runtimeKind": discovery.RuntimeKind, "state": discovery.State,
			"errorCode": nullableString(string(discovery.ErrorCode)), "detectedAt": discovery.DetectedAt.UTC(),
			"version": selectedVersion(discovery), "candidateCount": len(discovery.Candidates),
		}
	}
	instanceSummary := map[string]any{"freshness": "UNKNOWN", "errorCode": "DIAGNOSTICS_INSTANCE_READ_FAILED", "instances": []any{}}
	if runtimeErr == nil {
		listed, err := s.instances.ListInstances(ctx, "hermes")
		if err == nil {
			if len(listed.Instances) > MaxInstances {
				report.InstanceRecordsTruncated = len(listed.Instances) - MaxInstances
				listed.Instances = listed.Instances[:MaxInstances]
			}
			items := make([]map[string]any, 0, len(listed.Instances))
			for _, item := range listed.Instances {
				items = append(items, map[string]any{
					"instanceIdHash": hashValue(item.InstanceID), "nameHash": hashValue(item.Name),
					"default": item.Default, "protected": item.Protected, "availability": item.Availability,
					"lastSyncedAt": item.LastSyncedAt,
				})
				report.IdentifiersHashed += 2
			}
			instanceSummary = map[string]any{"freshness": listed.Freshness, "errorCode": nullableString(string(listed.ErrorCode)), "lastSyncedAt": listed.LastSyncedAt, "instances": items}
		}
	}
	operationItems := make([]map[string]any, 0)
	if values, err := s.operations.List(ctx, "", "", MaxOperations); err == nil {
		if len(values) > MaxOperations {
			report.OperationRecordsTruncated = len(values) - MaxOperations
			values = values[:MaxOperations]
		}
		for _, value := range values {
			operationItems = append(operationItems, map[string]any{
				"idHash": hashValue(value.ID), "type": value.Type, "targetType": value.TargetType,
				"targetIdHash": hashValue(value.TargetID), "status": value.Status, "stage": value.Stage,
				"progress": value.Progress, "errorCode": nullableString(string(value.ErrorCode)),
				"retryable": value.Retryable, "createdAt": value.CreatedAt.UTC(), "updatedAt": value.UpdatedAt.UTC(),
			})
			report.IdentifiersHashed += 2
		}
	}
	schemaVersion := 0
	if value, err := s.schema.SchemaVersion(ctx); err == nil {
		schemaVersion = value
	}
	logs := s.sanitizedLogs(now, report)
	files := map[string][]byte{}
	addJSON := func(name string, value any) error {
		body, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		files[name] = append(body, '\n')
		return nil
	}
	manifestFiles := []string{"manifest.json", "version.json", "node-summary.json", "runtime-summary.json", "instance-summary.json", "operations.json", "schema.json", "logs/node.ndjson", "redaction-report.json"}
	if err := addJSON("manifest.json", map[string]any{"bundleSchemaVersion": SchemaVersion, "createdAt": now, "files": manifestFiles, "bounds": map[string]any{"instanceRecords": MaxInstances, "operationRecords": MaxOperations, "logRecords": MaxLogRecords, "logWindowHours": int(LogWindow.Hours()), "uncompressedBytes": MaxUncompressedBytes, "bundleBytes": MaxBundleBytes}}); err != nil {
		return Bundle{}, err
	}
	if err := addJSON("version.json", map[string]any{"service": buildinfo.Service, "version": buildinfo.Version, "protocolVersion": buildinfo.ProtocolVersion}); err != nil {
		return Bundle{}, err
	}
	if err := addJSON("node-summary.json", map[string]any{"nodeIdHash": hashValue(s.node.ID), "hostnameHash": hashValue(s.node.Hostname), "platform": s.node.Platform, "architecture": s.node.Architecture, "nodeVersion": s.node.NodeVersion}); err != nil {
		return Bundle{}, err
	}
	report.IdentifiersHashed += 2
	if err := addJSON("runtime-summary.json", runtimeSummary); err != nil {
		return Bundle{}, err
	}
	if err := addJSON("instance-summary.json", instanceSummary); err != nil {
		return Bundle{}, err
	}
	if err := addJSON("operations.json", map[string]any{"operations": operationItems, "limit": MaxOperations}); err != nil {
		return Bundle{}, err
	}
	if err := addJSON("schema.json", map[string]any{"currentVersion": schemaVersion, "state": map[bool]string{true: "READY", false: "UNKNOWN"}[schemaVersion > 0]}); err != nil {
		return Bundle{}, err
	}
	files["logs/node.ndjson"] = logs
	if err := addJSON("redaction-report.json", report); err != nil {
		return Bundle{}, err
	}
	return encodeBundle(files, now)
}

func (s *Service) sanitizedLogs(now time.Time, report *redactionReport) []byte {
	path := filepath.Join(s.dataDir, "logs", "install.ndjson")
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || stat.ModTime().Before(now.Add(-LogWindow)) {
		return nil
	}
	start := stat.Size() - (MaxUncompressedBytes + 1)
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(file, MaxUncompressedBytes+1))
	if err != nil {
		return nil
	}
	lines := bytes.Split(body, []byte{'\n'})
	if start > 0 && len(lines) > 0 {
		report.LogRecordsTruncated++
		lines = lines[1:]
	}
	compacted := lines[:0]
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) != 0 {
			compacted = append(compacted, line)
		}
	}
	lines = compacted
	if len(lines) > MaxLogRecords {
		report.LogRecordsTruncated += len(lines) - MaxLogRecords
		lines = lines[len(lines)-MaxLogRecords:]
	}
	allowed := map[string]bool{"time": true, "level": true, "service": true, "version": true, "event": true, "runtimeKind": true, "stage": true, "status": true, "errorCode": true, "retryable": true, "durationMs": true, "count": true}
	var out bytes.Buffer
	for _, line := range lines {
		report.LogRecordsRead++
		if len(line) > MaxLogLineBytes {
			report.LogRecordsTruncated++
			continue
		}
		var source map[string]any
		if json.Unmarshal(line, &source) != nil {
			report.FieldsDropped++
			continue
		}
		observed, ok := source["time"].(string)
		observedAt, timeErr := time.Parse(time.RFC3339Nano, observed)
		if !ok || timeErr != nil || observedAt.Before(now.Add(-LogWindow)) || observedAt.After(now.Add(5*time.Minute)) {
			report.LogRecordsOutsideWindow++
			continue
		}
		clean := make(map[string]any)
		for key, value := range source {
			if !allowed[key] {
				report.FieldsDropped++
				continue
			}
			if text, ok := value.(string); ok {
				if unsafeText(text) {
					clean[key] = "[REDACTED]"
					report.ValuesRedacted++
				} else {
					clean[key] = truncate(text, 128)
				}
			} else if _, ok := value.(bool); ok {
				clean[key] = value
			} else if _, ok := value.(float64); ok {
				clean[key] = value
			} else {
				report.FieldsDropped++
			}
		}
		encoded, err := json.Marshal(clean)
		if err == nil {
			out.Write(encoded)
			out.WriteByte('\n')
			report.LogRecordsWritten++
		}
	}
	return out.Bytes()
}

func encodeBundle(files map[string][]byte, now time.Time) (Bundle, error) {
	names := make([]string, 0, len(files))
	total := 0
	for name, body := range files {
		names = append(names, name)
		total += len(body)
	}
	if total > MaxUncompressedBytes {
		return Bundle{}, ErrBundleTooLarge
	}
	sort.Strings(names)
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(now)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return Bundle{}, fmt.Errorf("create diagnostic entry: %w", err)
		}
		if _, err := writer.Write(files[name]); err != nil {
			return Bundle{}, fmt.Errorf("write diagnostic entry: %w", err)
		}
	}
	if err := archive.Close(); err != nil {
		return Bundle{}, fmt.Errorf("close diagnostic bundle: %w", err)
	}
	if output.Len() > MaxBundleBytes {
		return Bundle{}, ErrBundleTooLarge
	}
	return Bundle{Bytes: output.Bytes(), FileName: "YORVA-diagnostics-" + now.Format("20060102-150405") + ".zip", CreatedAt: now}, nil
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:8])
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func selectedVersion(value yorvaruntime.Discovery) any {
	if value.Selected == nil {
		return nil
	}
	return value.Selected.Version
}
func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
func unsafeText(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "canary") || strings.Contains(lower, "bearer ") || strings.Contains(lower, "authorization") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") || strings.Contains(lower, "token=") || strings.Contains(lower, "secret=") || strings.Contains(lower, "password=") || strings.Contains(lower, "cookie") || strings.Contains(lower, "sk-") || strings.Contains(value, `:\`) || strings.HasPrefix(value, "/home/") || strings.HasPrefix(value, "/Users/")
}
