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
	//go:embed uninstall-sindri.sh
	uninstallSh []byte
	//go:embed update-sindri.sh.tpl
	updateShTpl []byte
	//go:embed uninstall-sindri.cmd
	uninstallCmd []byte
	//go:embed update-sindri.cmd.tpl
	updateCmdTpl []byte
	//go:embed sindri.txt.tpl
	txtTpl []byte
)

// NewTarPrefixReader returns an io.Reader containing tar entries
// to prepend to a tar archive which help users to manage Sindri
// client-side.
func NewTarPrefixReader(r *http.Request) io.Reader {
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
		readmeName             = []byte("sindri.txt")
		uninstallSindriShName  = []byte("uninstall-sindri")
		updateSindriShName     = []byte("update-sindri")
		uninstallSindriCmdName = []byte("uninstall-sindri.cmd")
		updateSindriCmdName    = []byte("update-sindri.cmd")
		template               = func(b []byte) []byte {
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
				[]byte("__README_NAME__"), readmeName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UNINSTALL_SINDRI_SH_NAME__"), uninstallSindriShName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UPDATE_SINDRI_SH_NAME__"), updateSindriShName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UNINSTALL_SINDRI_CMD_NAME__"), uninstallSindriCmdName,
			)

			b = bytes.ReplaceAll(
				b,
				[]byte("__UPDATE_SINDRI_CMD_NAME__"), updateSindriCmdName,
			)

			return b
		}
	)

	go func() {
		defer pw.Close()

		if err := writeFile(
			"uninstall-sindri",
			template(uninstallSh),
			0755,
		); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if err := writeFile(
			"update-sindri",
			template(updateShTpl),
			0755,
		); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if err := writeFile(
			"uninstall-sindri.cmd",
			template(uninstallCmd),
			0755,
		); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if err := writeFile(
			"update-sindri.cmd",
			template(updateCmdTpl),
			0755,
		); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if err := writeFile(
			"sindri.txt",
			template(txtTpl),
			0644,
		); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	return pr
}
