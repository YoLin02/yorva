package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestAppSettingsRoundTripAndDelete(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	const key = "hermes.download-sources.test"
	payload := []byte(`{"pythonIndexUrl":"https://mirror.example/simple"}`)
	if err := database.PutAppSetting(ctx, key, payload, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := database.GetAppSetting(ctx, key)
	if err != nil || !found || string(loaded) != string(payload) {
		t.Fatalf("loaded=%q found=%v err=%v", loaded, found, err)
	}
	if err := database.DeleteAppSetting(ctx, key); err != nil {
		t.Fatal(err)
	}
	_, found, err = database.GetAppSetting(ctx, key)
	if err != nil || found {
		t.Fatalf("found=%v err=%v after delete", found, err)
	}
}
