package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
)

func TestChannelBindingStoresOnlySafeProjectionAndUpsertsOnePerType(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	installationID := seedInstallation(t, db)
	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{{NativeID: "default", Default: true}}, now); err != nil {
		t.Fatal(err)
	}
	instances, err := db.ListInstances(ctx, installationID)
	if err != nil || len(instances) != 1 || instances[0].Availability != instance.Available {
		t.Fatalf("instances = %#v, %v", instances, err)
	}
	id, err := NewChannelBindingID()
	if err != nil {
		t.Fatal(err)
	}
	checked := now
	value := channel.Binding{
		ID: id, InstanceID: instances[0].ID, Type: channel.Weixin, State: channel.Connected,
		AccountLabel: "微信账户", ExternalID: "wx-safe-id", LastCheckedAt: &checked,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.UpsertChannelBinding(ctx, value); err != nil {
		t.Fatal(err)
	}
	value.State = channel.Disconnected
	value.AccountLabel = ""
	value.ExternalID = ""
	value.UpdatedAt = now.Add(time.Minute)
	if err := db.UpsertChannelBinding(ctx, value); err != nil {
		t.Fatal(err)
	}
	rows, err := db.ListChannelBindings(ctx, instances[0].ID)
	if err != nil || len(rows) != 1 || rows[0].State != channel.Disconnected || rows[0].ExternalID != "" {
		t.Fatalf("bindings = %#v, %v", rows, err)
	}
	var metadata string
	if err := db.db.QueryRowContext(ctx, "SELECT metadata_json FROM channel_bindings WHERE id = ?", id).Scan(&metadata); err != nil || metadata != "{}" {
		t.Fatalf("metadata = %q, %v", metadata, err)
	}
}

func TestChannelBindingRejectsUnknownTypeAndCascadesWithInstance(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	installationID := seedInstallation(t, db)
	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{{NativeID: "default", Default: true}}, now); err != nil {
		t.Fatal(err)
	}
	instances, _ := db.ListInstances(ctx, installationID)
	if err := db.UpsertChannelBinding(ctx, channel.Binding{ID: "chb_bad", InstanceID: instances[0].ID, Type: "telegram", State: channel.Connected, CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("unknown channel type accepted")
	}
	value, err := channelBindingAt(instances[0].ID, channel.WeCom, channel.NotConfigured, now)
	if err != nil || db.UpsertChannelBinding(ctx, value) != nil {
		t.Fatalf("seed binding: %v", err)
	}
	if _, err := db.db.ExecContext(ctx, "DELETE FROM instances WHERE id = ?", instances[0].ID); err != nil {
		t.Fatal(err)
	}
	rows, err := db.ListChannelBindings(ctx, instances[0].ID)
	if err != nil || len(rows) != 0 {
		t.Fatalf("bindings after cascade = %#v, %v", rows, err)
	}
}
