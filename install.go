package main

import (
	"fmt"
	"os"
)

func installRelease(name, repo, assetPattern, dest string) error {
	fmt.Printf("Getting %s release...\n", name)
	release, err := GetRelease(repo)
	if err != nil {
		return err
	}
	if len(release.Assets) == 0 {
		return fmt.Errorf("no assets in %s release %s", repo, release.TagName)
	}

	assetName := fmt.Sprintf(assetPattern, release.TagName)
	var url string
	for _, a := range release.Assets {
		if a.Name == assetName {
			url = a.Url
			break
		}
	}
	if url == "" {
		return fmt.Errorf("asset %q not found in %s release %s", assetName, repo, release.TagName)
	}

	fmt.Printf("Extracting %s\n", name)
	f, err := download(url, assetName)
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	return Extract(Ctx, ZipFormat, f, dest)
}
