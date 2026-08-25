package managementhealth

import (
	"errors"
	"strings"
	"testing"
)

func TestProjectLogSnapshotRedactsAndSanitizes(t *testing.T) {
	input := strings.Join([]string{
		"Authorization: Bearer abcdefghijklmnopqrstuvwxyz",
		"api_key=sk-abcdefghijklmnopqrstuvwxyz012345",
		`failed path C:\Users\Alice\.hermes\config.yaml user=alice@example.com`,
		"\x1b[31m<script>alert(1)</script>\x1b[0m\u202esecret",
	}, "\r\n")

	got, err := ProjectLogSnapshot(LogGateway, []byte(input))
	if err != nil {
		t.Fatalf("ProjectLogSnapshot() error = %v", err)
	}
	joined := strings.Join(got.Lines, "\n")
	for _, prohibited := range []string{"abcdefghijklmnopqrstuvwxyz", "sk-", `C:\Users\Alice`, "alice@example.com", "<script>", "\x1b", "\u202e"} {
		if strings.Contains(joined, prohibited) {
			t.Fatalf("log projection contains %q: %q", prohibited, joined)
		}
	}
	for _, marker := range []string{"[REDACTED]", "[PATH]", "[ACCOUNT]", "‹script›"} {
		if !strings.Contains(joined, marker) {
			t.Fatalf("log projection missing %q: %q", marker, joined)
		}
	}
	if got.TotalBytes != len(joined) {
		t.Fatalf("TotalBytes = %d, joined bytes = %d", got.TotalBytes, len(joined))
	}
}

func TestProjectLogSnapshotBounds(t *testing.T) {
	t.Run("line count", func(t *testing.T) {
		got, err := ProjectLogSnapshot(LogAgent, []byte(strings.Repeat("line\n", MaxLogLines+20)))
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Lines) != MaxLogLines || !got.Truncated {
			t.Fatalf("lines = %d, truncated = %v", len(got.Lines), got.Truncated)
		}
	})

	t.Run("single line", func(t *testing.T) {
		got, err := ProjectLogSnapshot(LogErrors, []byte(strings.Repeat("界", MaxLogLineBytes)))
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Lines) != 1 || len(got.Lines[0]) > MaxLogLineBytes || !got.Truncated {
			t.Fatalf("line bytes = %d, truncated = %v", len(got.Lines[0]), got.Truncated)
		}
	})

	t.Run("total bytes", func(t *testing.T) {
		line := strings.Repeat("x", MaxLogLineBytes-1) + "\n"
		got, err := ProjectLogSnapshot(LogMCP, []byte(strings.Repeat(line, MaxLogLines)))
		if err != nil {
			t.Fatal(err)
		}
		if got.TotalBytes > MaxLogTotalBytes || !got.Truncated {
			t.Fatalf("total = %d, truncated = %v", got.TotalBytes, got.Truncated)
		}
	})

	t.Run("input bytes", func(t *testing.T) {
		_, err := ProjectLogSnapshot(LogDesktop, []byte(strings.Repeat("x", MaxLogInputBytes+1)))
		if !errors.Is(err, ErrLogSnapshotTooLarge) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestProjectLogSnapshotCategoryAllowlist(t *testing.T) {
	for _, category := range []LogCategory{LogAgent, LogErrors, LogGateway, LogGUI, LogDesktop, LogMCP} {
		if _, err := ProjectLogSnapshot(category, []byte("ok")); err != nil {
			t.Fatalf("category %q error = %v", category, err)
		}
	}
	if _, err := ProjectLogSnapshot("arbitrary/path", []byte("ok")); !errors.Is(err, ErrUnsupportedLogCategory) {
		t.Fatalf("unsupported category error = %v", err)
	}
	if SourceAcquisitionQualified {
		t.Fatal("source acquisition must remain unqualified")
	}
}
