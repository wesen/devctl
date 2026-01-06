package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/go-go-golems/devctl/pkg/logjs"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var version = "dev"

type options struct {
	scriptPath    string
	inputPath     string
	source        string
	format        string
	jsTimeout     string
	workers       int
	unsafeModules []string
}

func main() {
	opts := options{}

	rootCmd := &cobra.Command{
		Use:     "log-parse",
		Short:   "Parse log lines with a JavaScript module (goja-based)",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cmd, opts)
		},
	}

	cobra.CheckErr(logging.AddLoggingLayerToRootCommand(rootCmd, "log-parse"))

	rootCmd.Flags().StringVar(&opts.scriptPath, "js", "", "Path to JS parser module (required)")
	rootCmd.Flags().StringVar(&opts.inputPath, "input", "", "Input file path (default: stdin)")
	rootCmd.Flags().StringVar(&opts.source, "source", "", "Source label (default: input filename or stdin)")
	rootCmd.Flags().StringVar(&opts.format, "format", "ndjson", "Output format: ndjson|pretty")
	rootCmd.Flags().StringVar(&opts.jsTimeout, "js-timeout", "0", "Per-hook JS timeout (e.g. 50ms, 200ms)")
	rootCmd.Flags().IntVar(&opts.workers, "workers", 1, "Number of JS runtimes/workers (MVP: only 1 supported)")
	rootCmd.Flags().StringSliceVar(&opts.unsafeModules, "unsafe-modules", nil, "Opt-in unsafe modules (reserved; MVP ignores)")

	cobra.CheckErr(rootCmd.MarkFlagRequired("js"))
	cobra.CheckErr(rootCmd.Execute())
}

func run(ctx context.Context, cmd *cobra.Command, opts options) error {
	if opts.workers != 1 {
		return errors.New("MVP supports only --workers=1")
	}
	if opts.format != "ndjson" && opts.format != "pretty" {
		return errors.New("--format must be ndjson or pretty")
	}

	var r io.Reader
	var closer io.Closer
	if opts.inputPath != "" {
		f, err := os.Open(opts.inputPath)
		if err != nil {
			return err
		}
		r = f
		closer = f
	} else {
		r = os.Stdin
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	source := opts.source
	if source == "" {
		if opts.inputPath != "" {
			source = opts.inputPath
		} else {
			source = "stdin"
		}
	}

	m, err := logjs.LoadFromFile(ctx, opts.scriptPath, logjs.Options{HookTimeout: opts.jsTimeout})
	if err != nil {
		return err
	}
	defer func() { _ = m.Close(ctx) }()

	bw := bufio.NewWriter(cmd.OutOrStdout())
	defer func() { _ = bw.Flush() }()

	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	br := bufio.NewReader(r)
	var lineNumber int64
	for {
		line, err := br.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if line == "" && errors.Is(err, io.EOF) {
			break
		}

		lineNumber++
		ev, perr := m.ProcessLine(ctx, line, source, lineNumber)
		if perr != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "log-parse: line %d: %v\n", lineNumber, perr)
		}
		if ev != nil {
			switch opts.format {
			case "ndjson":
				if err := enc.Encode(ev); err != nil {
					return err
				}
			case "pretty":
				b, err := json.MarshalIndent(ev, "", "  ")
				if err != nil {
					return err
				}
				if _, err := bw.Write(append(b, '\n')); err != nil {
					return err
				}
			default:
				return errors.New("unsupported format")
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return nil
}
