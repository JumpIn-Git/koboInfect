package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetPlato() error {
	fmt.Println("Getting Plato release...")
	release, err := GetRelease("baskerville/plato")
	if err != nil {
		return err
	}

	if len(release.Assets) == 0 {
		return fmt.Errorf("no assets found in Plato release %s", release.TagName)
	}

	asset := &release.Assets[0]
	expectedName := fmt.Sprintf("plato-%s.zip", release.TagName)
	if asset.Name != expectedName {
		return fmt.Errorf("unexpected Plato release asset name: expected %q, got %q", expectedName, asset.Name)
	}

	src, err := saveArchive(asset.Url, "plato-*.zip", true)
	if err != nil {
		return err
	}
	defer os.Remove(src)

	fmt.Println("Extracting Plato")
	if err := Extract(Ctx, ZipFormat, src, filepath.Join(Root, ".adds", "plato")); err != nil {
		return fmt.Errorf("extracting plato: %w", err)
	}
	return nil
}
