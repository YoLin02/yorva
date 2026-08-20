package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestModelValidationOperationPassesAndProjectsLatestSummary(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	broker := events.NewBroker()
	inventory.WithEvents(broker)
	subscription := broker.Subscribe()
	defer subscription.Close()
	models := configuredFakeModels()
	models.validate = func(_ context.Context, nativeID, presetID, modelID string) yorvaruntime.ModelValidationResult {
		if nativeID != "coder" || presetID != "deepseek" || modelID != "deepseek-v4-pro" {
			t.Fatalf("validation target = %q %q %q", nativeID, presetID, modelID)
		}
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationPassed}
	}
	registerTestModels(t, inventory, models)
	instanceID := firstTestInstanceID(t, inventory)
	started, err := inventory.StartModelValidation(context.Background(), instanceID, "validate-pass")
	if err != nil || started.Operation.Type != operation.TypeModelValidate || started.Operation.TargetType != operation.TargetInstance || started.Operation.TargetID != instanceID {
		t.Fatalf("start = %#v %v", started, err)
	}
	completed := waitModelValidation(t, inventory, started.Operation.ID)
	if completed.Status != operation.StatusSucceeded || completed.ErrorCode != "" {
		t.Fatalf("completed = %#v", completed)
	}
	configuration, err := inventory.GetModelConfiguration(context.Background(), instanceID)
	if err != nil || configuration.Validation.State != "PASSED" || configuration.Validation.CompletedAt == nil {
		t.Fatalf("configuration = %#v %v", configuration, err)
	}
	models.config.ModelID = "changed-after-validation"
	configuration, err = inventory.GetModelConfiguration(context.Background(), instanceID)
	if err != nil || configuration.Validation.State != "NOT_RUN" || configuration.Validation.ErrorCode != "" {
		t.Fatalf("changed config retained stale validation = %#v %v", configuration.Validation, err)
	}

	foundCompleted := false
	deadline := time.After(2 * time.Second)
	for !foundCompleted {
		select {
		case event := <-subscription.Events:
			var payload events.OperationPayload
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				t.Fatal(err)
			}
			if payload.OperationID == completed.ID && payload.Status == string(operation.StatusSucceeded) {
				foundCompleted = true
				if strings.Contains(string(event.Data), "deepseek-v4-pro") || strings.Contains(string(event.Data), "secret") {
					t.Fatalf("event leaked model or credential material: %s", event.Data)
				}
			}
		case <-deadline:
			t.Fatal("validation completion event not published")
		}
	}
}

func TestModelValidationOperationNormalizesFailureAndUnknown(t *testing.T) {
	tests := []struct {
		name        string
		result      yorvaruntime.ModelValidationResult
		wantSummary string
	}{
		{name: "provider rejection", result: yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationFailed, ErrorCode: yorvaruntime.ErrorModelValidationFailed}, wantSummary: "FAILED"},
		{name: "timeout", result: yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorModelValidationTimedOut}, wantSummary: "UNKNOWN"},
		{name: "output limit", result: yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorModelValidationOutputLimit}, wantSummary: "UNKNOWN"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
			models := configuredFakeModels()
			models.validate = func(context.Context, string, string, string) yorvaruntime.ModelValidationResult { return test.result }
			registerTestModels(t, inventory, models)
			instanceID := firstTestInstanceID(t, inventory)
			started, err := inventory.StartModelValidation(context.Background(), instanceID, "validate-"+strings.ReplaceAll(test.name, " ", "-"))
			if err != nil {
				t.Fatal(err)
			}
			completed := waitModelValidation(t, inventory, started.Operation.ID)
			if completed.Status != operation.StatusFailed || completed.ErrorCode != test.result.ErrorCode || strings.Contains(completed.Message, "raw") {
				t.Fatalf("completed = %#v", completed)
			}
			configuration, err := inventory.GetModelConfiguration(context.Background(), instanceID)
			if err != nil || configuration.Validation.State != test.wantSummary || configuration.Validation.ErrorCode != test.result.ErrorCode {
				t.Fatalf("summary = %#v %v", configuration.Validation, err)
			}
		})
	}
}

func TestModelValidationCanCancelAndRejectsConcurrentStart(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := configuredFakeModels()
	entered := make(chan struct{})
	models.validate = func(ctx context.Context, _, _, _ string) yorvaruntime.ModelValidationResult {
		close(entered)
		<-ctx.Done()
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorModelValidationCancelled}
	}
	registerTestModels(t, inventory, models)
	instanceID := firstTestInstanceID(t, inventory)
	started, err := inventory.StartModelValidation(context.Background(), instanceID, "validate-cancel")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("validation did not enter adapter")
	}
	if _, err := inventory.StartModelValidation(context.Background(), instanceID, "validate-concurrent"); !errors.Is(err, yorvaruntime.ErrInstanceConfigConflict) {
		t.Fatalf("concurrent start error = %v", err)
	}
	cancelled, err := inventory.CancelModelValidation(context.Background(), started.Operation.ID)
	if err != nil || cancelled.Status != operation.StatusCancelled || cancelled.ErrorCode != yorvaruntime.ErrorModelValidationCancelled {
		t.Fatalf("cancelled = %#v %v", cancelled, err)
	}
	configuration, err := inventory.GetModelConfiguration(context.Background(), instanceID)
	if err != nil || configuration.Validation.State != "UNKNOWN" || configuration.Validation.ErrorCode != yorvaruntime.ErrorModelValidationCancelled {
		t.Fatalf("cancel summary = %#v %v", configuration.Validation, err)
	}
}

func TestModelValidationCancelRetriesAfterPendingWorkerTransition(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	now := inventory.now()
	pending := operation.Operation{
		ID: "op_validation_cancel_race", Type: operation.TypeModelValidate, TargetType: operation.TargetInstance, TargetID: "inst_public",
		Status: operation.StatusPending, Stage: operation.StagePreflight, IdempotencyKey: "cancel-race", CorrelationID: "cor_cancel_race",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := inventory.db.CreateOperation(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	running := pending
	running.Status = operation.StatusRunning
	running.Stage = operation.StageModelValidate
	running.StartedAt = &now
	running.UpdatedAt = now.Add(time.Millisecond)
	if err := inventory.db.UpdateOperation(context.Background(), pending, running); err != nil {
		t.Fatal(err)
	}

	cancelled, err := inventory.cancelModelValidationOperation(context.Background(), pending)
	if err != nil || cancelled.Status != operation.StatusCancelled || cancelled.ErrorCode != yorvaruntime.ErrorModelValidationCancelled {
		t.Fatalf("cancelled after CAS race = %#v %v", cancelled, err)
	}
}

func TestModelValidationRestartRecoveryBecomesUnknown(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := configuredFakeModels()
	registerTestModels(t, inventory, models)
	instanceID := firstTestInstanceID(t, inventory)
	now := inventory.now()
	orphan := operation.Operation{
		ID: "op_orphan_validation", Type: operation.TypeModelValidate, TargetType: operation.TargetInstance, TargetID: instanceID,
		Status: operation.StatusRunning, Stage: operation.StageModelValidate, IdempotencyKey: "orphan-validation", CorrelationID: "cor_orphan",
		SourcePin: modelConfigFingerprint("deepseek", "deepseek-v4-pro"), CreatedAt: now, StartedAt: &now, UpdatedAt: now,
	}
	if err := inventory.db.CreateOperation(context.Background(), orphan); err != nil {
		t.Fatal(err)
	}
	recovered, err := inventory.RecoverModelValidations(context.Background())
	if err != nil || len(recovered) != 1 || recovered[0].ErrorCode != yorvaruntime.ErrorOperationInterrupted {
		t.Fatalf("recovered = %#v %v", recovered, err)
	}
	configuration, err := inventory.GetModelConfiguration(context.Background(), instanceID)
	if err != nil || configuration.Validation.State != "UNKNOWN" || configuration.Validation.ErrorCode != yorvaruntime.ErrorOperationInterrupted {
		t.Fatalf("restart summary = %#v %v", configuration.Validation, err)
	}
}

func configuredFakeModels() *fakeModelConfigurator {
	return &fakeModelConfigurator{
		config:     yorvaruntime.ModelConfiguration{ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true},
		credential: yorvaruntime.ModelCredentialStatus{ProviderPresetID: "deepseek", Configured: true},
	}
}

func firstTestInstanceID(t *testing.T, inventory *InstanceInventory) string {
	t.Helper()
	listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
	if err != nil || len(listed.Instances) != 1 {
		t.Fatalf("instances = %#v %v", listed, err)
	}
	return listed.Instances[0].InstanceID
}

func waitModelValidation(t *testing.T, inventory *InstanceInventory, operationID string) operation.Operation {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		value, err := inventory.db.GetOperation(context.Background(), operationID)
		if err != nil {
			t.Fatal(err)
		}
		if operation.IsTerminal(value.Status) {
			return value
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("validation operation did not complete")
	return operation.Operation{}
}
