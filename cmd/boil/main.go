package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/frantjc/sindri/command"
	xerrors "github.com/frantjc/x/errors"
	xos "github.com/frantjc/x/os"
)

func main() {
	var (
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	)

	err := xerrors.Ignore(command.NewBoil().ExecuteContext(ctx), context.Canceled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	stop()
	xos.ExitFromError(err)
}
