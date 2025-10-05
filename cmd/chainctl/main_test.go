package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/internal/cli"
	telemetryinit "github.com/dobrovols/chainctl/internal/telemetry"
	secreterrors "github.com/dobrovols/chainctl/pkg/secrets"
)

type exitPanic struct{ code int }

func resetMainGlobals() {
	telemetryInit = telemetryinit.InitProvider
	rootCommand = cli.NewRootCommand
	osExit = os.Exit
}

func TestMainSuccess(t *testing.T) {
	t.Cleanup(func() {
		resetMainGlobals()
		os.Args = []string{"chainctl"}
	})

	var shutdownCalled bool
	telemetryInit = func(context.Context) (func(context.Context) error, error) {
		return func(context.Context) error {
			shutdownCalled = true
			return nil
		}, nil
	}

	var executed bool
	rootCommand = func() *cobra.Command {
		cmd := &cobra.Command{Run: func(cmd *cobra.Command, args []string) { executed = true }}
		return cmd
	}

	osExit = func(code int) {
		panic(exitPanic{code})
	}

	os.Args = []string{"chainctl"}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				t.Fatalf("unexpected exit code %d", ep.code)
			}
			panic(r)
		}
	}()

	main()

	if !executed {
		t.Fatalf("expected root command to execute")
	}
	if !shutdownCalled {
		t.Fatalf("expected telemetry shutdown to run")
	}
}

func TestMainTelemetryInitError(t *testing.T) {
	t.Cleanup(func() {
		resetMainGlobals()
		os.Args = []string{"chainctl"}
	})

	telemetryInit = func(context.Context) (func(context.Context) error, error) {
		return nil, errors.New("init failed")
	}

	rootCommand = func() *cobra.Command {
		return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
	}

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
		w.Close()
	}()

	os.Args = []string{"chainctl"}

	main()

	w.Close()
	out, _ := io.ReadAll(r)
	r.Close()

	if !bytes.Contains(out, []byte("failed to initialize telemetry")) {
		t.Fatalf("expected telemetry init error in stderr, got %q", string(out))
	}
}

func TestMainSecretErrorExit(t *testing.T) {
	t.Cleanup(func() {
		resetMainGlobals()
		os.Args = []string{"chainctl"}
	})

	telemetryInit = func(context.Context) (func(context.Context) error, error) {
		return nil, nil
	}

	rootCommand = func() *cobra.Command {
		return &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error {
			return secreterrors.NewError(secreterrors.ErrCodeValidation, errors.New("boom"))
		}}
	}

	var exitCode int
	osExit = func(code int) {
		panic(exitPanic{code: code})
	}

	os.Args = []string{"chainctl"}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				exitCode = ep.code
				return
			}
			panic(r)
		}
	}()

	main()

	if exitCode != secreterrors.ErrCodeValidation {
		t.Fatalf("expected exit code %d, got %d", secreterrors.ErrCodeValidation, exitCode)
	}
}
