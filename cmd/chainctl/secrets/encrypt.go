package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	secrethandler "github.com/dobrovols/chainctl/pkg/secrets"
)

var (
	isTerminal   = term.IsTerminal
	readPassword = term.ReadPassword
	stdinFD      = func() int { return int(os.Stdin.Fd()) }
)

type encryptOptions struct {
	InputPath  string
	OutputPath string
	Passphrase string
	Overwrite  bool
	Format     string
}

// NewEncryptCommand returns the `chainctl encrypt-values` command implementation.
func NewEncryptCommand() *cobra.Command {
	opts := encryptOptions{}

	cmd := &cobra.Command{
		Use:   "encrypt-values",
		Short: "Encrypt a plaintext Helm values file for secure distribution",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runEncryptCommand(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.InputPath, "input", "", "Path to plaintext values file")
	cmd.Flags().StringVar(&opts.OutputPath, "output", "", "Destination for encrypted values")
	cmd.Flags().StringVar(&opts.Passphrase, "passphrase", "", "Encryption passphrase (interactive prompt if omitted)")
	cmd.Flags().BoolVar(&opts.Overwrite, "confirm", false, "Allow overwriting an existing output file")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format: text or json")

	return cmd
}

func runEncryptCommand(cmd *cobra.Command, opts encryptOptions) error {
	if err := validateEncryptPaths(opts); err != nil {
		return err
	}
	format, err := normalizeEncryptFormat(opts.Format)
	if err != nil {
		return err
	}
	passphrase, err := resolvePassphrase(cmd.ErrOrStderr(), opts.Passphrase)
	if err != nil {
		return err
	}

	result, err := secrethandler.EncryptFile(secrethandler.EncryptOptions{
		InputPath:  opts.InputPath,
		OutputPath: opts.OutputPath,
		Passphrase: passphrase,
		Overwrite:  opts.Overwrite,
	})
	if err != nil {
		return err
	}

	return renderEncryptResult(cmd, format, result)
}

func validateEncryptPaths(opts encryptOptions) error {
	if strings.TrimSpace(opts.InputPath) == "" {
		return secrethandler.NewError(secrethandler.ErrCodeValidation, errors.New("input path is required"))
	}
	if strings.TrimSpace(opts.OutputPath) == "" {
		return secrethandler.NewError(secrethandler.ErrCodeValidation, errors.New("output path is required"))
	}
	return nil
}

func normalizeEncryptFormat(format string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	if normalized == "" {
		normalized = "text"
	}
	if normalized != "text" && normalized != "json" {
		return "", secrethandler.NewError(secrethandler.ErrCodeValidation, fmt.Errorf("unsupported format %q", normalized))
	}
	return normalized, nil
}

func resolvePassphrase(writer io.Writer, provided string) (string, error) {
	if strings.TrimSpace(provided) != "" {
		return provided, nil
	}
	passphrase, err := promptForPassphrase(writer)
	if err != nil {
		return "", secrethandler.NewError(secrethandler.ErrCodeValidation, err)
	}
	return passphrase, nil
}

func renderEncryptResult(cmd *cobra.Command, format string, result *secrethandler.EncryptResult) error {
	if result == nil {
		return secrethandler.NewError(secrethandler.ErrCodeEncryption, errors.New("encryption result is nil"))
	}
	switch format {
	case "json":
		payload := map[string]string{
			"outputPath": result.OutputPath,
			"checksum":   result.Checksum,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return secrethandler.NewError(secrethandler.ErrCodeEncryption, err)
		}
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Encrypted values written to %s\nChecksum: %s\n", result.OutputPath, result.Checksum)
	}
	return nil
}

func promptForPassphrase(writer io.Writer) (string, error) {
	fd := stdinFD()
	if !isTerminal(fd) {
		return "", errors.New("passphrase must be provided via --passphrase in non-interactive mode")
	}

	fmt.Fprint(writer, "Enter passphrase: ")
	pass1, err := readPassword(fd)
	fmt.Fprintln(writer)
	if err != nil {
		return "", err
	}

	fmt.Fprint(writer, "Confirm passphrase: ")
	pass2, err := readPassword(fd)
	fmt.Fprintln(writer)
	if err != nil {
		zero(pass1)
		return "", err
	}

	if !bytes.Equal(pass1, pass2) {
		zero(pass1)
		zero(pass2)
		return "", errors.New("passphrases do not match")
	}

	passphrase := string(pass1)
	zero(pass1)
	zero(pass2)
	return passphrase, nil
}

func zero(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
