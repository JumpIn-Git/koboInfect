package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func SaveNm(merge bool) (string, error) {
	release, err := GetRelease("pgaskin/nickelmenu")
	if err != nil {
		return "", err
	}

	asset := release.Assets[0]
	if asset.Name != "KoboRoot.tgz" {
		return "", UnexpectedRelease
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

func unpackTGZ(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", src, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to read gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		name := strings.TrimPrefix(h.Name, "./")
		target := filepath.Join(dst, name)

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", filepath.Dir(target), err)
			}
			out, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("failed to write %s: %w", target, err)
			}
			out.Close()
		}
	}
	return nil
}
