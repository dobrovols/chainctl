package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dobrovols/chainctl/internal/cli"
	telemetryinit "github.com/dobrovols/chainctl/internal/telemetry"
	secreterrors "github.com/dobrovols/chainctl/pkg/secrets"
)

var (
	telemetryInit = telemetryinit.InitProvider
	rootCommand   = cli.NewRootCommand
	osExit        = os.Exit
)

func main() {
	ctx := context.Background()
	shutdown, err := telemetryInit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize telemetry: %v\n", err)
	}
	if shutdown != nil {
		cleanupCtx, cancel := context.WithTimeout(ctx, telemetryinit.ShutdownTimeout)
		defer func() {
			defer cancel()
			if err := shutdown(cleanupCtx); err != nil {
				fmt.Fprintf(os.Stderr, "telemetry shutdown error: %v\n", err)
			}
		}()
	}

	cmd := rootCommand()
	if err := cmd.Execute(); err != nil {
		var encErr *secreterrors.Error
		if errors.As(err, &encErr) {
			osExit(encErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}
