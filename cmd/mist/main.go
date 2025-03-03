package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/frantjc/sindri/command"
	_ "github.com/frantjc/sindri/steamapp"
	_ "github.com/frantjc/sindri/steamworkshopitem"
	_ "github.com/frantjc/sindri/thunderstore"
	xerrors "github.com/frantjc/x/errors"
	xos "github.com/frantjc/x/os"
)

func main() {
	var (
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		cmd       = command.AddCommon(command.NewMist(), SemVer())
	)

	err := xerrors.Ignore(cmd.ExecuteContext(ctx), context.Canceled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	stop()
	xos.ExitFromError(err)
}
