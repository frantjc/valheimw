package controller

import (
	"context"
	"io"

	"github.com/frantjc/sindri/internal/api/v1alpha1"
)

type ImageScanner interface {
	Scan(context.Context, io.Reader) ([]v1alpha1.Vulnerability, error)
}
