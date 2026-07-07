package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func installRelease(ctx context.Context, name, repo, assetPattern string, saveFunc func(ctx context.Context, f *os.File) error) error {
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

	return saveFunc(ctx, f)
}

func GetKoreader(ctx context.Context) error {
	return installRelease(ctx, "KOReader", "koreader/koreader", "koreader-kobo-%s.zip", func(ctx context.Context, f *os.File) error {
		return ExtractPrefix(ctx, Zip, f, Prefixes{
			"koreader/": filepath.Join(Root, ".adds", "koreader"), // Zip has koreader.png for KFmon
		})
	})
}

func GetPlato(ctx context.Context) error {
	return installRelease(ctx, "Plato", "baskerville/plato", "plato-%s.zip", func(ctx context.Context, f *os.File) error {
		return Extract(ctx, Zip, f, filepath.Join(Root, ".adds", "plato"))
	})
}
