package main

import (
	"fmt"
	"path/filepath"
)

func SaveNm(merge bool) (string, error) {
	release, err := GetRelease("pgaskin/nickelmenu")
	if err != nil {
		return "", err
	}

	if len(release.Assets) == 0 {
		return "", fmt.Errorf("no assets found in nickelmenu release %s", release.TagName)
	}

	asset := release.Assets[0]
	if asset.Name != "KoboRoot.tgz" {
		return "", fmt.Errorf("unexpected nickelmenu release asset name: expected KoboRoot.tgz, got %q", asset.Name)
	}

	if !merge {
		_, err := saveArchive(asset.Url, filepath.Join(Root, ".kobo", "KoboRoot.tgz"), false)
		return "", err
	}
	f, err := saveArchive(asset.Url, "nm-*.tgz", true)
	if err != nil {
		return "", err
	}
	return f, nil
}
