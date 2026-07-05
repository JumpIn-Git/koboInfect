package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
		return nil, fmt.Errorf("%s: failed to decode release: %w", repo, err)
	}

	return &release, nil
}

func saveArchive(url, path string, tmp bool) (string, error) {
	var f *os.File
	var err error
	if !tmp {
		f, err = os.Open(path)
		if err != nil {
			return "", err
		}
	} else {
		f, err = os.CreateTemp("", path)
		if err != nil {
			return "", err
		}
	}
	defer f.Close()

	resp, err := Get(url)
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to save zip: %w", err)
	}
	return f.Name(), nil
}

func Get(url string) (*http.Response, error) {
	resp, err := Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("couldn't download %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("response status for %s: %d", url, resp.StatusCode)
	}
	return resp, nil
}
