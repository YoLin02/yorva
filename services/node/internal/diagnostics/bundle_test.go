package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeRuntimeReader struct{ discovery yorvaruntime.Discovery }

func (f fakeRuntimeReader) Detect(context.Context, yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	return f.discovery, nil
}

type fakeInstanceReader struct{ list app.InstanceList }

func (f fakeInstanceReader) ListInstances(context.Context, string) (app.InstanceList, error) {
	return f.list, nil
}

type fakeOperationReader struct{ values []operation.Operation }

func (f fakeOperationReader) List(context.Context, string, string, int) ([]operation.Operation, error) {
	return f.values, nil
}

type fakeSchemaReader int

func (f fakeSchemaReader) SchemaVersion(context.Context) (int, error) { return int(f), nil }

func TestBuildProducesFixedSanitizedBundle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "logs"), 0o700); err != nil {
		t.Fatal(err)
	}
	canary := "SECRET_CANARY_sk-never-export"
	logLine := fmt.Sprintf(`{"time":"2026-09-04T07:00:00Z","level":"ERROR","service":"yorvad","event":"Bearer %s","detail":"%s","authorization":"%s","stage":"backup.publish"}`+"\n", canary, canary, canary)
	if err := os.WriteFile(filepath.Join(root, "logs", "install.ndjson"), []byte(logLine), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 4, 7, 1, 2, 0, time.UTC)
	selected := &yorvaruntime.Candidate{Version: "1.2.3", State: yorvaruntime.DiscoverySupported}
	service := New(
		node.Node{ID: "node-secret", Hostname: "Alice-PC", Platform: "windows", Architecture: "amd64", NodeVersion: "0.4.0"},
		fakeRuntimeReader{yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported, Selected: selected, DetectedAt: now}},
		fakeInstanceReader{app.InstanceList{Freshness: "LIVE", Instances: []app.InstanceView{{InstanceID: "inst-secret", Name: "private-profile", Availability: "AVAILABLE"}}}},
		fakeOperationReader{[]operation.Operation{{ID: "op-secret", TargetID: "inst-secret", Type: operation.TypeInstanceStart, TargetType: operation.TargetInstance, Status: operation.StatusSucceeded, UpdatedAt: now, CreatedAt: now}}},
		fakeSchemaReader(17), root,
	)
	service.now = func() time.Time { return now }
	bundle, err := service.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.FileName != "YORVA-diagnostics-20260904-070102.zip" {
		t.Fatalf("filename = %q", bundle.FileName)
	}
	if bytes.Contains(bundle.Bytes, []byte(canary)) || bytes.Contains(bundle.Bytes, []byte("Alice-PC")) || bytes.Contains(bundle.Bytes, []byte("private-profile")) {
		t.Fatal("compressed bundle contains a secret canary or raw identifier")
	}
	files := unzipFiles(t, bundle.Bytes)
	want := []string{"instance-summary.json", "logs/node.ndjson", "manifest.json", "node-summary.json", "operations.json", "redaction-report.json", "runtime-summary.json", "schema.json", "version.json"}
	for _, name := range want {
		if _, ok := files[name]; !ok {
			t.Fatalf("missing %s", name)
		}
	}
	joined := bytes.Join(mapValues(files), nil)
	for _, forbidden := range []string{canary, "Alice-PC", "private-profile", "inst-secret", "op-secret", "authorization", "detail"} {
		if bytes.Contains(joined, []byte(forbidden)) {
			t.Fatalf("bundle leaked %q", forbidden)
		}
	}
	if !bytes.Contains(files["logs/node.ndjson"], []byte(`"event":"[REDACTED]"`)) {
		t.Fatalf("log not redacted: %s", files["logs/node.ndjson"])
	}
	var schema map[string]any
	if err := json.Unmarshal(files["schema.json"], &schema); err != nil {
		t.Fatal(err)
	}
	if schema["currentVersion"] != float64(17) || schema["state"] != "READY" {
		t.Fatalf("schema = %#v", schema)
	}
}

func TestBuildTruncatesLargeLogHistory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "logs"), 0o700); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	for index := 0; index < MaxLogRecords+40; index++ {
		fmt.Fprintf(&body, `{"time":"2026-09-04T07:00:00Z","level":"INFO","event":"event-%d"}`+"\n", index)
	}
	if err := os.WriteFile(filepath.Join(root, "logs", "install.ndjson"), body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	operations := make([]operation.Operation, MaxOperations+5)
	instances := make([]app.InstanceView, MaxInstances+5)
	service := New(node.Node{}, fakeRuntimeReader{}, fakeInstanceReader{app.InstanceList{Instances: instances}}, fakeOperationReader{operations}, fakeSchemaReader(17), root)
	service.now = func() time.Time { return time.Date(2026, 9, 4, 7, 1, 2, 0, time.UTC) }
	bundle, err := service.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	files := unzipFiles(t, bundle.Bytes)
	lines := bytes.Split(bytes.TrimSpace(files["logs/node.ndjson"]), []byte{'\n'})
	if len(lines) > MaxLogRecords {
		t.Fatalf("log records = %d", len(lines))
	}
	var report redactionReport
	if err := json.Unmarshal(files["redaction-report.json"], &report); err != nil {
		t.Fatal(err)
	}
	if report.LogRecordsTruncated == 0 {
		t.Fatal("expected truncation to be reported")
	}
	if report.OperationRecordsTruncated != 5 || report.InstanceRecordsTruncated != 5 {
		t.Fatalf("record truncation = %#v", report)
	}
}

func unzipFiles(t *testing.T, body []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string][]byte)
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		value, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		result[file.Name] = value
	}
	return result
}

func mapValues(values map[string][]byte) [][]byte {
	result := make([][]byte, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
