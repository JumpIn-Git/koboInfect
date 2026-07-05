package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Client = http.Client{Timeout: 1 * time.Minute}
var UnexpectedRelease = errors.New("unexpected release")

type Asset struct {
	Name string `json:"name"`
	Url  string `json:"browser_download_url"`
}
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []Asset
}

func GetRelease(repo string) (*Release, error) {
	resp, err := Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("%s: failed to decode release JSON: %w", repo, err)
	}

	return &release, nil
}

func downloadZip(url string) (path string, err error) {
	tmp, err := os.CreateTemp("", "tmp-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmp.Close()

	resp, err := Get(url)
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("failed to download zip: %w", err)
	}
	return tmp.Name(), nil
}

func extractZipPrefix(zipPath, prefix, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel == "" || rel == "/" {
			if f.FileInfo().IsDir() {
				continue
			}
			// file matches prefix exactly — write to dst directly
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", filepath.Dir(dst), err)
			}
			out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", dst, err)
			}
			rc, err := f.Open()
			if err != nil {
				out.Close()
				return fmt.Errorf("failed to open %s in zip: %w", f.Name, err)
			}
			_, err = io.Copy(out, rc)
			out.Close()
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to write %s: %w", dst, err)
			}
			continue
		}
		rel = strings.TrimPrefix(rel, "/")
		fPath := filepath.Join(dst, rel)
		if !strings.HasPrefix(fPath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path: %s", fPath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", fPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", filepath.Dir(fPath), err)
		}

		out, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", fPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return fmt.Errorf("failed to open %s in zip: %w", f.Name, err)
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", fPath, err)
		}
	}

	return nil
}

func ExtractZip(url, prefix, dst string) error {
	zipPath, err := downloadZip(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer os.Remove(zipPath)
	return extractZipPrefix(zipPath, prefix, dst)
}

func ExtractZipPrefixes(url string, prefixDsts map[string]string) error {
	zipPath, err := downloadZip(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer os.Remove(zipPath)
	for prefix, dst := range prefixDsts {
		if err := extractZipPrefix(zipPath, prefix, dst); err != nil {
			return fmt.Errorf("extracting %q: %w", prefix, err)
		}
	}
	return nil
}

func Get(url string) (*http.Response, error) {
	resp, err := Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("couldn't download %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status for %s: %d", url, resp.StatusCode)
	}
	return resp, nil
}
