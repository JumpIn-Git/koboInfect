package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v3"
)

var Client = http.Client{Timeout: 1 * time.Minute}

type Asset struct {
	Name string `json:"name"`
	Url  string `json:"browser_download_url"`
}
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func GetRelease(repo string) (*Release, error) {
	resp, err := Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("%s: decoding release: %w\n", repo, err)
	}

	return &release, nil
}

func download(url, pattern, name string) (*os.File, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, err
	}

	resp, err := Get(url)
	if err != nil {
		os.Remove(f.Name())
		f.Close()
		return nil, err
	}
	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		fmt.Sprintf("downloading %s", name),
	)

	if _, err := io.Copy(io.MultiWriter(f, bar), resp.Body); err != nil {
		os.Remove(f.Name())
		f.Close()
		return nil, fmt.Errorf("saving archive: %w\n", err)
	}

	if _, err := f.Seek(0, 0); err != nil {
		os.Remove(f.Name())
		f.Close()
		return nil, err
	}
	return f, nil
}

func downloadTo(url, path, name string) error {
	resp, err := Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		fmt.Sprintf("downloading %s", name),
	)

	if _, err = io.Copy(io.MultiWriter(f, bar), resp.Body); err != nil {
		return err
	}
	return f.Close()
}

func Get(url string) (*http.Response, error) {
	resp, err := Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w\n", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("response status for %s: %d\n", url, resp.StatusCode)
	}
	return resp, nil
}
