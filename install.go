package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func installRelease(ctx context.Context, name, repo, assetPattern string, saveFunc func(ctx context.Context, url string) error) error {
	fmt.Printf("Getting %s release...\n", name)
	release, err := GetRelease(repo)
	if err != nil {
		return err
	}
	if len(release.Assets) == 0 {
		return fmt.Errorf("no assets in %s release %s", repo, release.TagName)
	}

	assetName := assetPattern
	if strings.Contains(assetPattern, "%s") {
		assetName = fmt.Sprintf(assetPattern, release.TagName)
	}
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

	return saveFunc(ctx, url)
}

func GetKoreader(ctx context.Context) error {
	return installRelease(ctx, "KOReader", "koreader/koreader", "koreader-kobo-%s.zip", func(ctx context.Context, url string) error {
		f, err := download(url, "koreader-kobo-*.zip")
		if err != nil {
			return err
		}
		defer f.Close()
		defer os.Remove(f.Name())
		return ExtractPrefix(ctx, Zip, f, Prefixes{
			"koreader/": filepath.Join(Root, ".adds", "koreader"),
		})
	})
}

func GetPlato(ctx context.Context) error {
	return installRelease(ctx, "Plato", "baskerville/plato", "plato-%s.zip", func(ctx context.Context, url string) error {
		f, err := download(url, "plato-*.zip")
		if err != nil {
			return err
		}
		defer f.Close()
		defer os.Remove(f.Name())
		return Extract(ctx, Zip, f, filepath.Join(Root, ".adds", "plato"))
	})
}

func GetNM(merge bool) (*os.File, error) {
	var out *os.File
	err := installRelease(context.Background(), "NickelMenu", "pgaskin/nickelmenu", "KoboRoot.tgz", func(ctx context.Context, url string) error {
		if !merge {
			return downloadTo(url, filepath.Join(Root, ".kobo", "KoboRoot.tgz"))
		}
		var err error
		out, err = download(url, "nm-*.tgz")
		return err
	})
	return out, err
}
