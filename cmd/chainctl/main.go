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

func main() {
	ctx := context.Background()
	shutdown, err := telemetryinit.InitProvider(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize telemetry: %v\n", err)
	}
	if shutdown != nil {
		cleanupCtx, cancel := context.WithTimeout(ctx, telemetryinit.ShutdownTimeout)
		defer func() {
			defer cancel()
			shutdown(cleanupCtx)
		}()
	}

	cmd := cli.NewRootCommand()
	if err := cmd.Execute(); err != nil {
		var encErr *secreterrors.Error
		if errors.As(err, &encErr) {
			os.Exit(encErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
