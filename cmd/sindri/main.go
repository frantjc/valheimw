package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	errorcode "github.com/frantjc/go-error-code"
	"github.com/frantjc/go-fn"
	"github.com/frantjc/sindri/command"
)

func main() {
	var (
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err       error
	)

	if err := command.NewSindri().ExecuteContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	stop()
	os.Exit(errorcode.ExitCode(fn.Ternary(errors.Is(err, context.Canceled), nil, err)))
}
