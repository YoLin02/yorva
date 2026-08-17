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
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
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
	if err := hermes.Register(registry); err != nil {
		return fmt.Errorf("register Hermes Runtime descriptor: %w", err)
	}
	database, err := sqlite.Open(ctx, message.DataDir)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer database.Close()

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
	host := hermes.NewHostInstaller(message.DataDir).WithLogger(logger).WithEmbeddedSource(message.HermesEmbeddedSourcePath)
	nodeHost := hermes.NewNodeHost(message.DataDir, message.HermesNodeArchivePath, message.HermesNpmArchivePath)
	installs := app.NewRuntimeInstall(discovery, database).WithLogger(logger).WithHost(host, database, localNode.ID).WithPrerequisite(app.HermesPrerequisiteHost{Host: nodeHost})
	if _, err := installs.InterruptStale(ctx); err != nil {
		logger.Warn("failed to interrupt stale install operations", "error", err)
	}
	server := &http.Server{
		Handler:           httpapi.NewHandler(message.Token, localNode, events.NewBroker(), discovery, installs, message.DataDir),
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
		parentDone <- monitorParent(stdin)
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
	Type string `json:"type"`
}

func monitorParent(r *bufio.Reader) error {
	line, err := r.ReadBytes('\n')
	if errors.Is(err, io.EOF) && len(line) == 0 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read parent control: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var control parentControl
	if err := decoder.Decode(&control); err != nil || control.Type != "shutdown" {
		return errors.New("invalid parent control message")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("invalid parent control message")
	}
	return nil
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
