package x11

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	xslice "github.com/frantjc/x/slice"
)

const (
	EnvVarXAuthority = "XAUTHORITY"
)

type XAuth struct {
	Path       string
	XAuthority string
}

func (x *XAuth) List(ctx context.Context, displays ...*Display) ([]XAuthorityEntry, error) {
	if err := x.init(); err != nil {
		return nil, err
	}

	var (
		args = append([]string{"list"}, xslice.Map(displays, func(d *Display, _ int) string {
			return d.String()
		})...)
		cmd    = exec.CommandContext(ctx, x.Path, args...)
		pr, pw = io.Pipe()
	)
	if x.XAuthority != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", EnvVarXAuthority, x.XAuthority))
	}
	cmd.Stdout = pw

	go func() {
		_ = pw.CloseWithError(cmd.Run())
	}()

	var (
		scanner           = bufio.NewScanner(pr)
		xauthorityEntries = []XAuthorityEntry{}
	)
	for scanner.Scan() {
		xauthorityEntry, err := ParseXAuthorityEntry(scanner.Text())
		if err != nil {
			return nil, err
		}

		xauthorityEntries = append(xauthorityEntries, *xauthorityEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return xauthorityEntries, nil
}

func (x *XAuth) Add(ctx context.Context, xauthorityEntry *XAuthorityEntry) error {
	if err := x.init(); err != nil {
		return err
	}

	var (
		cmd = exec.CommandContext(ctx, x.Path, "-n", "add", xauthorityEntry.Display.String(), xauthorityEntry.Proto, xauthorityEntry.Cookie)
	)
	if x.XAuthority != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", EnvVarXAuthority, x.XAuthority))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (x *XAuth) Remove(ctx context.Context, display *Display) error {
	if err := x.init(); err != nil {
		return err
	}

	var (
		cmd = exec.CommandContext(ctx, x.Path, "remove", display.String())
	)
	if x.XAuthority != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", EnvVarXAuthority, x.XAuthority))
	}

	return cmd.Run()
}

func (x *XAuth) init() error {
	if x.Path == "" {
		x.Path = "xauth"
	}

	if x.XAuthority != "" {
		var err error
		if x.XAuthority, err = filepath.Abs(x.XAuthority); err != nil {
			return err
		}

		f, err := os.Create(x.XAuthority)
		if err != nil {
			return err
		}

		if err = f.Close(); err != nil {
			return err
		}
	}

	return nil
}
