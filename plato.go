package main

import (
	"fmt"
	"path/filepath"
)

func GetPlato() error {
	fmt.Println("Getting Plato release...")
	release, err := GetRelease("baskerville/plato")
	if err != nil {
		return err
	}

	asset := &release.Assets[0]
	if asset.Name != fmt.Sprintf("plato-%s.zip", release.TagName) {
		return UnexpectedRelease
	}

	fmt.Println("Extracting Plato")
	if err := ExtractZip(asset.Url, "", filepath.Join(Root, ".adds", "plato")); err != nil {
		return fmt.Errorf("extracting plato: %w", err)
	}
	return nil
}
