package valheim

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ReadSeed(opts *Opts) (string, error) {
	return ReadSaveDirSeed(opts.SaveDir, opts.World)
}

func ReadSaveDirSeed(savedir, world string) (string, error) {
	f, err := os.Open(filepath.Join(savedir, "worlds_local", world+".fwl"))
	if err != nil {
		return "", err
	}

	return ReadWorldSeed(f, world)
}

var (
	SeedLength = 10
)

func ReadWorldSeed(r io.Reader, world string) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	parts := bytes.Split(b, []byte(world+"\n"))
	if len(parts) < 2 {
		return "", fmt.Errorf("unable to parse world %s seed", world)
	}

	part := parts[1]
	if len(part) < SeedLength {
		return "", fmt.Errorf("unable to parse world %s seed", world)
	}

	return string(part[:SeedLength]), nil
}
