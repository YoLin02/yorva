package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/applog"
	"github.com/YoLin02/yorva/services/node/internal/bootstrap"
	"github.com/YoLin02/yorva/services/node/internal/buildinfo"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/install"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/backupmanagement"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
	"github.com/YoLin02/yorva/services/node/internal/secrets"
	"github.com/YoLin02/yorva/services/node/internal/transport/httpapi"
)

type Streams struct {
	Stdin  io.ReadCloser
	Stdout io.Writer
	Stderr io.Writer
}

func Run(ctx context.Context, args []string, streams Streams) error {
	if len(args) != 1 || args[0] != "--bootstrap-stdio" {
		return errors.New("yorvad requires --bootstrap-stdio")
	}
	defer streams.Stdin.Close()

	stdin := bootstrap.NewReader(streams.Stdin)
	message, err := bootstrap.ReadMessage(stdin, buildinfo.ProtocolVersion)
	if err != nil {
		return fmt.Errorf("read bootstrap configuration: %w", err)
	}

	logger, closeLog := applog.New(streams.Stderr, message.DataDir)
	defer closeLog()
	registry := yorvaruntime.NewRegistry()
	database, err := sqlite.Open(ctx, message.DataDir)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer database.Close()
	destinationGrants, err := backupmanagement.NewDestinationRegistry(message.Token)
	if err != nil {
		return fmt.Errorf("initialize native backup destination authority: %w", err)
	}
	mcpManager, err := hermes.NewProductionProfileMCPManager()
	if err != nil {
		return fmt.Errorf("initialize YORVA MCP test server: %w", err)
	}
	defer mcpManager.Close()
	bindings := hermes.ManagementBindings{
		MCPRead:    mcpManager,
		MCPMutate:  mcpManager,
		BackupRead: sqlite.NewRuntimeBackupReader(database, string(hermes.Kind)),
	}
	if runtime.GOOS == "windows" {
		if recoverErr := hermes.RecoverInterruptedRestores(ctx); recoverErr != nil {
			return fmt.Errorf("recover interrupted Hermes Restore: %w", recoverErr)
		}
		secretStore, secretErr := secrets.New(message.DataDir)
		if secretErr != nil {
			return fmt.Errorf("initialize backup SecretStore: %w", secretErr)
		}
		useDeviceKey := func(ctx context.Context, reference string, use func([]byte) error) error {
			ref := secrets.Reference(reference)
			metadata, inspectErr := secretStore.Inspect(ctx, ref)
			if inspectErr != nil {
				return inspectErr
			}
			if !metadata.Configured {
				generateErr := secretStore.Generate(ctx, ref, func() ([]byte, error) {
					_, identity, keyErr := backupmanagement.GenerateX25519Identity()
					return identity, keyErr
				})
				if generateErr != nil && !errors.Is(generateErr, secrets.ErrAlreadyExists) {
					return generateErr
				}
			}
			return secretStore.Get(ctx, ref, use)
		}
		insertBackup := func(ctx context.Context, entry hermes.BackupIndexEntry) error {
			return database.InsertVerifiedRuntimeBackup(ctx, sqlite.RuntimeBackupIndexEntry{
				ID: entry.ID, RuntimeInstallationID: entry.RuntimeInstallationID,
				FormatVersion: entry.FormatVersion, RuntimeVersion: entry.RuntimeVersion,
				ArtifactPath: entry.ArtifactPath, SizeBytes: entry.SizeBytes,
				ChecksumSHA256: entry.ChecksumSHA256, State: sqlite.RuntimeBackupAvailable,
				KeyMode: sqlite.RuntimeBackupDeviceKey, KeyRef: entry.KeyRef,
				CreatedAt: entry.CreatedAt, VerifiedAt: entry.VerifiedAt, UpdatedAt: entry.VerifiedAt,
			})
		}
		getBackup := func(ctx context.Context, backupID string) (hermes.BackupIndexEntry, error) {
			entry, readErr := database.GetRuntimeBackupByID(ctx, backupID)
			if readErr != nil {
				return hermes.BackupIndexEntry{}, readErr
			}
			return hermes.BackupIndexEntry{
				ID: entry.ID, RuntimeInstallationID: entry.RuntimeInstallationID,
				FormatVersion: entry.FormatVersion, RuntimeVersion: entry.RuntimeVersion,
				ArtifactPath: entry.ArtifactPath, SizeBytes: entry.SizeBytes,
				ChecksumSHA256: entry.ChecksumSHA256, KeyRef: entry.KeyRef,
				CreatedAt: entry.CreatedAt, VerifiedAt: entry.VerifiedAt,
			}, nil
		}
		deleteBackup := func(ctx context.Context, backupID string) error {
			return database.DeleteRuntimeBackupByID(ctx, backupID)
		}
		backupManager := hermes.NewRuntimeBackupManager(destinationGrants, message.DataDir, useDeviceKey, insertBackup, getBackup, deleteBackup)
		bindings.BackupMutate = backupManager
		bindings.Restore = backupManager
	}
	if err := hermes.RegisterConfigured(registry, bindings); err != nil {
		return fmt.Errorf("register Hermes Runtime descriptor: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("resolve hostname: %w", err)
	}
	localNode, err := database.LoadOrCreateNode(ctx, node.LocalMetadata{
		Name:         hostname,
		Hostname:     hostname,
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		NodeVersion:  buildinfo.Version,
	})
	if err != nil {
		return fmt.Errorf("initialize local node: %w", err)
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	defer listener.Close()

	port, err := listenerPort(listener.Addr())
	if err != nil {
		return err
	}

	requestCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	discovery := app.NewRuntimeDiscovery(registry, logger)
	sourceSettings := downloadsources.NewService(database)
	host := hermes.NewHostInstaller(message.DataDir).WithLogger(logger).WithEmbeddedSource(message.HermesEmbeddedSourcePath).WithEmbeddedPython(message.HermesPythonArchivePath).WithDownloadSources(sourceSettings)
	installGate := install.NewGateHolder()
	managedRoot := ""
	if root, err := install.DefaultManagedRoot(); err != nil {
		installGate.Set(install.GateBlockedUnsafe)
		logger.Error("managed Hermes root is unavailable", "error", err, "gate", installGate.Get())
	} else {
		managedRoot = root
		if _, recErr := install.Recover(context.Background(), root, installGate); recErr != nil {
			logger.Error("install recovery failed", "error", recErr, "gate", installGate.Get())
		}
	}
	nodeHost := hermes.NewNodeHost(message.DataDir, message.HermesNodeArchivePath, message.HermesNpmArchivePath).WithDownloadSources(sourceSettings)
	broker := events.NewBroker()
	installs := app.NewRuntimeInstall(discovery, database).WithLogger(logger).WithHost(host, database, localNode.ID).WithPrerequisite(app.HermesPrerequisiteHost{Host: nodeHost}).WithEvents(broker).WithInstallGate(installGate).WithManagedRoot(managedRoot)
	if _, err := installs.InterruptStale(ctx); err != nil {
		logger.Warn("failed to interrupt stale install operations", "error", err)
	}
	instances := app.NewInstanceInventory(discovery, database, app.HermesProfileSource{}, localNode.ID).WithMutator(app.HermesProfileSource{}).WithEvents(broker)
	if _, err := instances.RecoverStale(ctx); err != nil {
		logger.Warn("failed to recover stale instance operations", "error", err)
	}
	if _, err := instances.RecoverModelValidations(ctx); err != nil {
		logger.Warn("failed to recover stale model validation operations", "error", err)
	}
	if _, err := instances.RecoverLifecycle(ctx); err != nil {
		logger.Warn("failed to recover stale lifecycle operations", "error", err)
	}
	if _, err := instances.RecoverChannels(ctx); err != nil {
		logger.Warn("failed to recover stale channel operations", "error", err)
	}
	server := &http.Server{
		Handler:           httpapi.NewHandler(message.Token, localNode, broker, discovery, installs, instances, message.DataDir, sourceSettings),
		BaseContext:       func(net.Listener) context.Context { return requestCtx },
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	if err := bootstrap.WriteHandshake(streams.Stdout, bootstrap.Handshake{
		ProtocolVersion: buildinfo.ProtocolVersion,
		Port:            port,
		PID:             os.Getpid(),
	}); err != nil {
		return err
	}

	parentDone := make(chan error, 1)
	go func() {
		parentDone <- monitorParent(stdin, streams.Stdout, destinationGrants)
	}()
	parentFinished := false
	defer func() {
		_ = streams.Stdin.Close()
		if !parentFinished {
			<-parentDone
		}
	}()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	logger.Info("daemon listening", "address", listener.Addr().String(), "dataDirConfigured", message.DataDir != "")

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case parentErr := <-parentDone:
		parentFinished = true
		if parentErr != nil {
			logger.Warn("parent control channel ended", "error", parentErr)
		}
	case <-ctx.Done():
	}

	cancelRequests()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}

type parentControl struct {
	Type           string `json:"type"`
	RequestID      string `json:"requestId,omitempty"`
	DestinationRef string `json:"destinationRef,omitempty"`
	RuntimeID      string `json:"runtimeId,omitempty"`
	Path           string `json:"path,omitempty"`
}

type parentControlAck struct {
	Type           string `json:"type"`
	RequestID      string `json:"requestId"`
	DestinationRef string `json:"destinationRef"`
	Accepted       bool   `json:"accepted"`
}

func monitorParent(r *bufio.Reader, stdout io.Writer, destinations *backupmanagement.DestinationRegistry) error {
	seenRequests := make(map[string]struct{})
	seenRequestOrder := make([]string, 0, 64)
	for {
		line, err := r.ReadSlice('\n')
		if errors.Is(err, io.EOF) && len(line) == 0 {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read parent control: %w", err)
		}

		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var control parentControl
		if err := decoder.Decode(&control); err != nil {
			return errors.New("invalid parent control message")
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return errors.New("invalid parent control message")
		}

		switch control.Type {
		case "shutdown":
			if control.RequestID != "" || control.DestinationRef != "" || control.RuntimeID != "" || control.Path != "" {
				return errors.New("invalid parent control message")
			}
			return nil
		case "backup_destination.grant":
			if control.RequestID == "" || control.DestinationRef == "" || control.RuntimeID == "" || control.Path == "" {
				return errors.New("invalid parent control message")
			}
			if _, duplicate := seenRequests[control.RequestID]; duplicate {
				return errors.New("duplicate parent control request")
			}
			if len(seenRequestOrder) == 64 {
				delete(seenRequests, seenRequestOrder[0])
				copy(seenRequestOrder, seenRequestOrder[1:])
				seenRequestOrder = seenRequestOrder[:63]
			}
			seenRequests[control.RequestID] = struct{}{}
			seenRequestOrder = append(seenRequestOrder, control.RequestID)
			accepted := destinations != nil && destinations.Grant(control.DestinationRef, control.RuntimeID, control.Path) == nil
			if err := json.NewEncoder(stdout).Encode(parentControlAck{
				Type: "backup_destination.ack", RequestID: control.RequestID,
				DestinationRef: control.DestinationRef, Accepted: accepted,
			}); err != nil {
				return fmt.Errorf("write parent control acknowledgement: %w", err)
			}
		default:
			return errors.New("invalid parent control message")
		}
	}
}

func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

func listenerPort(address net.Addr) (int, error) {
	_, portText, err := net.SplitHostPort(address.String())
	if err != nil {
		return 0, fmt.Errorf("parse listener address: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, fmt.Errorf("parse listener port: %w", err)
	}
	return port, nil
}
