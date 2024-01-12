package valheim

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func OpenFWL(savedir, world string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(savedir, "worlds_local", world+".fwl"))
}

func OpenDB(savedir, world string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(savedir, "worlds_local", world+".db"))
}

func ReadWorldSeed(savedir, world string) (string, error) {
	r, err := OpenFWL(savedir, world)
	if err != nil {
		return "", err
	}
	defer r.Close()

	return ReadSeed(r, world)
}

const (
	SeedLength = 10
)

func ReadSeed(r io.Reader, world string) (string, error) {
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
