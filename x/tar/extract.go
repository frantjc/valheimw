package xtar

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Extract reads the tarball from r and writes it into dir.
// Copied and modified from golang.org/x/build/internal/untar.
func Extract(r io.Reader, dir string) error {
	var (
		t0      = time.Now()
		madeDir = map[string]bool{}
	)

	tr := tar.NewReader(r)
	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if !validRelPath(f.Name) {
			return fmt.Errorf("tar contained invalid name: %s", f.Name)
		}

		var (
			rel  = filepath.FromSlash(f.Name)
			abs  = filepath.Join(dir, rel)
			fi   = f.FileInfo()
			mode = fi.Mode()
		)
		switch {
		case mode.IsRegular():
			// Make the directory. This is redundant because it should
			// already be made by a directory entry in the tar
			// beforehand. Thus, don't check for errors; the next
			// write will fail with the same error.
			dir := filepath.Dir(abs)
			if !madeDir[dir] {
				if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
					return err
				}
				madeDir[dir] = true
			}
			if runtime.GOOS == "darwin" && mode&0o111 != 0 {
				// The darwin kernel caches binary signatures
				// and SIGKILLs binaries with mismatched
				// signatures. Overwriting a binary with
				// O_TRUNC does not clear the cache, rendering
				// the new copy unusable. Removing the original
				// file first does clear the cache. See #54132.
				if err := os.Remove(abs); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
			}

			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return err
			}

			//nolint:gosec
			n, err := io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}

			if err != nil {
				return fmt.Errorf("writing to %s: %v", abs, err)
			}

			if n != f.Size {
				return fmt.Errorf("only wrote %d bytes to %s; expected %d", n, abs, f.Size)
			}

			modTime := f.ModTime
			if modTime.After(t0) {
				// Clamp modtimes at system time. See
				// golang.org/issue/19062 when clock on
				// buildlet was behind the gitmirror server
				// doing the git-archive.
				modTime = t0
			}
			if !modTime.IsZero() {
				_ = os.Chtimes(abs, modTime, modTime)
			}
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return err
			}

			madeDir[abs] = true
		default:
			return fmt.Errorf("tar file entry %s contained unsupported file type: %v", f.Name, mode)
		}
	}

	return nil
}

func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
