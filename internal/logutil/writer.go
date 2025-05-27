package logutil

import (
	"log/slog"
)

type LogWriter struct {
	*slog.Logger
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.Info(string(p))
	return len(p), nil
}
