package main

import (
	"path/filepath"
)

func GetKoreader() error {
	return installRelease("KOReader", "koreader/koreader", "koreader-kobo-%s.zip", filepath.Join(Root, ".adds", "koreader"))
}
