package valheim

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// findWorldFile locates a world data file with the given extension under
// savedir/worlds_local. Valheim has changed the on-disk layout of world
// files across versions (e.g. worlds_local/<world>.fwl vs
// worlds_local/<world>/_main.0.fwl2), so rather than hardcoding a single
// path, glob for candidates and pick the best match.
func findWorldFile(savedir, world, ext string) (string, error) {
	flat, err := filepath.Glob(filepath.Join(savedir, "worlds_local", world+"."+ext+"*"))
	if err != nil {
		return "", err
	}
	candidates := append([]string{}, flat...)

	nested, err := filepath.Glob(filepath.Join(savedir, "worlds_local", world, "*."+ext+"*"))
	if err != nil {
		return "", err
	}
	candidates = append(candidates, nested...)

	if len(candidates) == 0 {
		return "", fmt.Errorf("unable to find world %s .%s file in %s", world, ext, savedir)
	}

	return candidates[0], nil
}

func OpenFWL(savedir, world string) (io.ReadCloser, error) {
	path, err := findWorldFile(savedir, world, "fwl")
	if err != nil {
		return nil, err
	}

	return os.Open(path)
}

func OpenDB(savedir, world string) (io.ReadCloser, error) {
	path, err := findWorldFile(savedir, world, "db")
	if err != nil {
		return nil, err
	}

	return os.Open(path)
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
