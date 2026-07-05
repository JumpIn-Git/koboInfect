package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetKoreader() error {
	fmt.Println("Getting koreader release...")
	release, err := GetRelease("koreader/koreader")
	if err != nil {
		return err
	}
	if len(release.Assets) == 0 {
		return fmt.Errorf("no assets found in koreader release %s", release.TagName)
	}

	var url string
	expectedName := fmt.Sprintf("koreader-kobo-%s.zip", release.TagName)
	for _, asset := range release.Assets {
		if asset.Name == expectedName {
			url = asset.Url
			break
		}
	}
	if url == "" {
		return fmt.Errorf("could not find asset %q in koreader release %s", expectedName, release.TagName)
	}

	fmt.Println("Extracting koreader")
	src, err := saveArchive(url, "koreader-*.zip", true)
	if err != nil {
		return err
	}
	defer os.Remove(src)
	if err := Extract(Ctx, ZipFormat, src, filepath.Join(Root, ".adds", "koreader")); err != nil {
		return fmt.Errorf("extracting koreader: %w", err)
	}
	return nil
}
