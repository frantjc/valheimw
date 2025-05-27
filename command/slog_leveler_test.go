package command_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/frantjc/sindri/command"
	"github.com/spf13/pflag"
)

func TestSlogLeveler_AddFlags(t *testing.T) {
	var (
		slogLeveler = new(command.SlogLeveler)
		flagSet     = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	if err := os.Unsetenv("DEBUG"); err != nil {
		t.Fatalf("failed to unset DEBUG environment variable: %v", err)
	}

	slogLeveler.AddFlags(flagSet)

	if slogLeveler.Level() != slog.LevelError {
		t.Fatalf("expected level %v, got %v", slog.LevelError, slogLeveler.Level())
	}

	if err := flagSet.Parse([]string{"--debug"}); err != nil {
		t.Fatalf("failed to set debug flag: %v", err)
	}

	if slogLeveler.Level() != slog.LevelDebug {
		t.Fatalf("expected level %v, got %v", slog.LevelDebug, slogLeveler.Level())
	}

	if err := flagSet.Parse([]string{"--quiet"}); err != nil {
		t.Fatalf("failed to set quiet flag: %v", err)
	}

	if slogLeveler.Level() != slog.LevelError {
		t.Fatalf("expected level %v, got %v", slog.LevelError, slogLeveler.Level())
	}

	if err := flagSet.Parse([]string{"-v"}); err != nil {
		t.Fatalf("failed to set V flag: %v", err)
	}

	if slogLeveler.Level() != slog.LevelWarn {
		t.Fatalf("expected level %v, got %v", slog.LevelWarn, slogLeveler.Level())
	}
}
