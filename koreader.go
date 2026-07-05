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
	for _, asset := range release.Assets {
		if asset.Name == fmt.Sprintf("koreader-kobo-%s.zip", release.TagName) {
			url = asset.Url
			break
		}
	}
	if url == "" {
		return UnexpectedRelease
	}

	fmt.Println("Extracting koreader")
	if err := ExtractZip(url, "koreader/", filepath.Join(Root, ".adds", "koreader")); err != nil {
		return err
	}
	return nil
}
