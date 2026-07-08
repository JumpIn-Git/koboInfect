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
		return fmt.Errorf("no assets in %s release %s\n", repo, release.TagName)
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
		return fmt.Errorf("asset %q not found in %s release %s\n", assetName, repo, release.TagName)
	}

	return saveFunc(ctx, url)
}

func GetKoreader(ctx context.Context, root string) error {
	err := installRelease(ctx, "KOReader", "koreader/koreader", "koreader-kobo-%s.zip", func(ctx context.Context, url string) error {
		f, err := download(url, "koreader-kobo-*.zip", "KOReader")
		if err != nil {
			return err
		}
		defer f.Close()
		defer os.Remove(f.Name())
		return ExtractPrefix(ctx, Zip, f, Prefixes{
			"koreader/": filepath.Join(root, ".adds", "koreader"),
		})
	})
	if err != nil {
		return err
	}

	if err := optWrite(filepath.Join(root, ".adds", "nm", "koreader"),
		`menu_item : main : KOReader : cmd_spawn : quiet : exec /mnt/onboard/.adds/koreader/koreader.sh`); err != nil {
		return fmt.Errorf("failed to write to .adds/nm/koreader: %w\n", err)
	}
	return nil
}

func GetPlato(ctx context.Context, root string) error {
	err := installRelease(ctx, "Plato", "baskerville/plato", "plato-%s.zip", func(ctx context.Context, url string) error {
		f, err := download(url, "plato-*.zip", "Plato")
		if err != nil {
			return err
		}
		defer f.Close()
		defer os.Remove(f.Name())
		return Extract(ctx, Zip, f, filepath.Join(root, ".adds", "plato"))
	})
	if err != nil {
		return err
	}

	if err := optWrite(filepath.Join(root, ".adds", "nm", "plato"),
		`menu_item : main : Plato : cmd_spawn : quiet : exec /mnt/onboard/.adds/plato/plato.sh`); err != nil {
		return fmt.Errorf("failed to write to .adds/nm/plato: %w\n", err)
	}
	return nil
}

func GetNM(ctx context.Context, merge bool, root string) (*os.File, error) {
	var out *os.File
	err := installRelease(ctx, "NickelMenu", "pgaskin/nickelmenu", "KoboRoot.tgz", func(ctx context.Context, url string) error {
		if !merge {
			return downloadTo(url, filepath.Join(root, ".kobo", "KoboRoot.tgz"), "NickelMenu")
		}
		var err error
		out, err = download(url, "nm-*.tgz", "NickelMenu")
		return err
	})
	return out, err
}
