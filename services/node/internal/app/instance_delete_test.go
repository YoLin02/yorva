package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
)

func TestStartDeleteProtectsDefaultAndConfirmation(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	var defaultID, coderID string
	for _, item := range listed.Instances {
		if item.Name == "default" {
			defaultID = item.InstanceID
		}
		if item.Name == "coder" {
			coderID = item.InstanceID
		}
	}
	if _, err := inventory.StartDelete(context.Background(), defaultID, "default", "del-1"); !errors.Is(err, ErrInstanceProtected) {
		t.Fatalf("default delete = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("protected delete started a process")
	}
	if _, err := inventory.StartDelete(context.Background(), coderID, "wrong", "del-2"); !errors.Is(err, ErrInstanceConfirmationMismatch) {
		t.Fatalf("mismatch = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("mismatch started a process")
	}
}

func TestStartDeleteConvergesToMissingTombstone(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	var coderID string
	for _, item := range listed.Instances {
		if item.Name == "coder" {
			coderID = item.InstanceID
		}
	}
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-ok")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), started.Operation.ID)
		if err == nil && got.Status == operation.StatusSucceeded {
			row, err := inventory.GetInstance(context.Background(), coderID)
			if err != nil || row.Availability != instance.Missing || row.InstanceID != coderID {
				t.Fatalf("tombstone = %#v %v", row, err)
			}
			return
		}
		if err == nil && got.Status == operation.StatusFailed {
			t.Fatalf("delete failed: %#v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("delete operation did not succeed")
}

func TestStartDeleteDisappearanceRaceIsSuccess(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	var coderID string
	for _, item := range listed.Instances {
		if item.Name == "coder" {
			coderID = item.InstanceID
		}
	}
	source.removeProfile("coder")
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-race")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), started.Operation.ID)
		if err == nil && got.Status == operation.StatusSucceeded {
			_, _ = mutator.snapshot()
			row, err := inventory.GetInstance(context.Background(), coderID)
			if err != nil || row.Availability != instance.Missing {
				t.Fatalf("race tombstone = %#v %v", row, err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("disappearance race did not converge")
}
