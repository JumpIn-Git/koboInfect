package main

import (
	"fmt"
	"io"
	"os"
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

	resp, err := Get(asset.Url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if !merge {
		f, err := os.Create(filepath.Join(Root, ".kobo", "KoboRoot.tgz"))
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(f, resp.Body); err != nil {
			return "", err
		}
		return "", nil
	}
	tmp, err := os.CreateTemp("", "nm-*.tgz")
	if err != nil {
		return "", fmt.Errorf("failed to make tmp dir: %w", err)
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}
