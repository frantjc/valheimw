package logutil

import "github.com/go-logr/logr"

type LogWriter struct {
	logr.Logger
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.Info(string(p))
	return len(p), nil
}
