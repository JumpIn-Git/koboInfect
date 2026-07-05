package main

import (
	"fmt"
	"path/filepath"
)

func GetKoreader() error {
	fmt.Println("Getting koreader release...")
	release, err := GetRelease("koreader/koreader")
	if err != nil {
		return err
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
	if err := ExtractZip(url, "koreader/", filepath.Join(Root, ".adds", "koreader")); err != nil {
		return err
	}
	return nil
}
