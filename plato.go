package main

import (
	"path/filepath"
)

func GetPlato() error {
	return installRelease("Plato", "baskerville/plato", "plato-%s.zip", filepath.Join(Root, ".adds", "plato"))
}
