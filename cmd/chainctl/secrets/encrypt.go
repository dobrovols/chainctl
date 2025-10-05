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

// NewEncryptCommand returns the `chainctl encrypt-values` command implementation.
func NewEncryptCommand() *cobra.Command {
	var inputPath string
	var outputPath string
	var passphrase string
	var overwrite bool
	var format string

	cmd := &cobra.Command{
		Use:   "encrypt-values",
		Short: "Encrypt a plaintext Helm values file for secure distribution",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			if inputPath == "" {
				return secrethandler.NewError(secrethandler.ErrCodeValidation, errors.New("input path is required"))
			}
			if outputPath == "" {
				return secrethandler.NewError(secrethandler.ErrCodeValidation, errors.New("output path is required"))
			}

			format = strings.ToLower(format)
			if format == "" {
				format = "text"
			}
			if format != "text" && format != "json" {
				return secrethandler.NewError(secrethandler.ErrCodeValidation, fmt.Errorf("unsupported format %q", format))
			}

			if passphrase == "" {
				var err error
				passphrase, err = promptForPassphrase(cmd.ErrOrStderr())
				if err != nil {
					return secrethandler.NewError(secrethandler.ErrCodeValidation, err)
				}
			}

			result, err := secrethandler.EncryptFile(secrethandler.EncryptOptions{
				InputPath:  inputPath,
				OutputPath: outputPath,
				Passphrase: passphrase,
				Overwrite:  overwrite,
			})
			if err != nil {
				return err
			}

			switch format {
			case "json":
				payload := map[string]string{
					"outputPath": result.OutputPath,
					"checksum":   result.Checksum,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if encodeErr := enc.Encode(payload); encodeErr != nil {
					return secrethandler.NewError(secrethandler.ErrCodeEncryption, encodeErr)
				}
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "Encrypted values written to %s\nChecksum: %s\n", result.OutputPath, result.Checksum)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&inputPath, "input", "", "Path to plaintext values file")
	cmd.Flags().StringVar(&outputPath, "output", "", "Destination for encrypted values")
	cmd.Flags().StringVar(&passphrase, "passphrase", "", "Encryption passphrase (interactive prompt if omitted)")
	cmd.Flags().BoolVar(&overwrite, "confirm", false, "Allow overwriting an existing output file")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")

	return cmd
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
