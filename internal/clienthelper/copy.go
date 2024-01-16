package clienthelper

import (
	"archive/tar"
	"bytes"
	_ "embed"
	"io"
	"net/http"
	"time"
)

var (
	//go:embed uninstall.sh.tpl
	uninstallShTpl []byte
	//go:embed update.sh.tpl
	updateShTpl []byte
	//go:embed uninstall.cmd.tpl
	uninstallCmdTpl []byte
	//go:embed update.cmd.tpl
	updateCmdTpl []byte
	//go:embed readme.txt.tpl
	readmeTxtTpl []byte
)

var (
	readmeTxtName    = []byte("sindri.txt")
	uninstallShName  = []byte("uninstall-sindri")
	updateShName     = []byte("update-sindri")
	uninstallCmdName = []byte("uninstall-sindri.cmd")
	updateCmdName    = []byte("update-sindri.cmd")
)

func CopyWithTarPrefix(dst io.Writer, src io.Reader, r *http.Request) (int64, error) {
	n, err := io.Copy(dst, newTarPrefixReader(r))
	if err != nil {
		return n, err
	}

	m, err := io.Copy(dst, src)
	return n + m, err
}

// newTarPrefixReader returns an io.Reader containing tar entries
// to prepend to a tar archive which help users to manage Sindri
// client-side.
func newTarPrefixReader(r *http.Request) io.Reader {
	var (
		host     = []byte(r.Header.Get("X-Forwarded-Host"))
		protocol = []byte(r.Header.Get("X-Forwarded-Proto"))
	)

	if len(host) == 0 {
		host = []byte(r.Host)
	}

	if len(protocol) == 0 {
		protocol = []byte(r.Header.Get("X-Forwarded-Scheme"))
		if len(protocol) == 0 {
			protocol = []byte("http")
		}
	}

	var (
		pr, pw    = io.Pipe()
		tw        = tar.NewWriter(pw)
		now       = time.Now()
		writeFile = func(name string, contents []byte, mode int64) error {
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     name,
				Size:     int64(len(contents)),
				Mode:     mode,
				ModTime:  now,
			}); err != nil {
				return err
			}

			if _, err := tw.Write(contents); err != nil {
				return err
			}

			return tw.Flush()
		}
		template = func(b []byte) []byte {
			b = bytes.ReplaceAll(
				b,
				[]byte("__HOST__"), host,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__PROTOCOL__"), protocol,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__README_TXT_NAME__"), readmeTxtName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UNINSTALL_SH_NAME__"), uninstallShName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UPDATE_SH_NAME__"), updateShName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UNINSTALL_CMD_NAME__"), uninstallCmdName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UPDATE_CMD_NAME__"), updateCmdName,
			)

			return b
		}
	)

	go func() {
		defer pw.Close()

		if err := func() error {
			if err := writeFile(
				string(uninstallShName),
				template(uninstallShTpl),
				0755,
			); err != nil {
				return err
			}

			if err := writeFile(
				string(updateShName),
				template(updateShTpl),
				0755,
			); err != nil {
				return err
			}

			if err := writeFile(
				string(uninstallCmdName),
				template(uninstallCmdTpl),
				0755,
			); err != nil {
				return err
			}

			if err := writeFile(
				string(updateCmdName),
				template(updateCmdTpl),
				0755,
			); err != nil {
				return err
			}

			return writeFile(
				string(readmeTxtName),
				template(readmeTxtTpl),
				0644,
			)
		}(); err != nil {
			_ = pw.CloseWithError(err)
		}
	}()

	return pr
}
