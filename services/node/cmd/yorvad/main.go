package main

import (
	"context"
	"fmt"
	"os"

	"github.com/YoLin02/yorva/services/node/internal/daemon"
)

func main() {
	ctx, stop := daemon.SignalContext(context.Background())
	defer stop()

	if err := daemon.Run(ctx, os.Args[1:], daemon.Streams{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "yorvad startup failed: %v\n", err)
		os.Exit(1)
	}
}
