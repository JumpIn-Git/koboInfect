package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func SaveNm(merge bool) (*os.File, error) {
	release, err := GetRelease("pgaskin/nickelmenu")
	if err != nil {
		return nil, err
	}
	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("no assets in nickelmenu release %s", release.TagName)
	}

	asset := release.Assets[0]
	if asset.Name != "KoboRoot.tgz" {
		return nil, fmt.Errorf("unexpected nickelmenu asset: expected KoboRoot.tgz, got %q", asset.Name)
	}

	if !merge {
		return nil, downloadTo(asset.Url, filepath.Join(Root, ".kobo", "KoboRoot.tgz"))
	}
	return download(asset.Url, "nm-*.tgz")
}
