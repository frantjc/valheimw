package xzip

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func Extract(zr *zip.Reader, dir string) error {
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		//nolint:gosec
		path := filepath.Join(dir, f.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		t, err := os.Create(path)
		if err != nil {
			return err
		}
		defer t.Close()

		fr, err := f.Open()
		if err != nil {
			return err
		}
		defer fr.Close()

		//nolint:gosec
		if _, err = io.Copy(t, fr); err != nil {
			return err
		}
	}

	return nil
}
