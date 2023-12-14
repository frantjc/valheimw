package sindri

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	xio "github.com/frantjc/sindri/x/io"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// Logger is an alias to logr.Logger in case
// the logging library is desired to be swapped out.
type Logger = logr.Logger

// WithLogger returns a Context from the parent Context
// with the given Logger inside of it.
func WithLogger(ctx context.Context, log Logger) context.Context {
	return logr.NewContext(ctx, log)
}

// LoggerFrom returns a Logger embedded within the given Context
// or a no-op Logger if no such Logger exists.
func LoggerFrom(ctx context.Context) Logger {
	return logr.FromContextOrDiscard(ctx)
}

// NewLogger creates a new Logger.
func NewLogger() Logger {
	zapLogger, err := zap.NewProduction(
		zap.AddCallerSkip(1),
	)
	if err != nil {
		panic(err)
	}

	return zapr.NewLogger(zapLogger)
}

var (
	errStderr = fmt.Errorf("stderr")
)

// LogExec redirects a command's stdout and stderr
// to the given Logger.
func LogExec(log Logger, cmd *exec.Cmd) {
	log = log.WithValues("bin", cmd.Path)

	cmd.Stdout = xio.WriterFunc(func(b []byte) (int, error) {
		for _, p := range bytes.Split(b, []byte("\n")) {
			if len(p) > 0 {
				log.Info(strings.TrimSpace(string(p)))
			}
		}

		return len(b), nil
	})

	cmd.Stderr = xio.WriterFunc(func(b []byte) (int, error) {
		for _, p := range bytes.Split(b, []byte("\n")) {
			if len(p) > 0 {
				log.Error(errStderr, strings.TrimSpace(string(p)))
			}
		}

		return len(b), nil
	})
}
