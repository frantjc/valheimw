package testdata

import (
	_ "embed"
)

var (
	//go:embed alpine.tar
	Alpine []byte
)
