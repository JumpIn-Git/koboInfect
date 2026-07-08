package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

func handleFirmware(ctx context.Context, root string) error {
	fmt.Println("checking for firmware upgrade...")
	updateUrl, err := UpgradeCheck(root)
	if err != nil {
		return fmt.Errorf("firmware check: %w\n", err)
	}

	upgrading := updateUrl != ""

	nmArchive, err := GetNM(ctx, upgrading, root)
	if err != nil {
		return fmt.Errorf("saving NickelMenu: %w\n", err)
	}
	if nmArchive != nil {
		defer nmArchive.Close()
		defer os.Remove(nmArchive.Name())
	}

	if !upgrading {
		fmt.Println("Done.")
		return nil
	}

	combined, err := os.MkdirTemp("", "combined*")
	if err != nil {
		return fmt.Errorf("creating combined firmware root dir: %w\n", err)
	}
	defer os.RemoveAll(combined)

	nmStat, err := nmArchive.Stat()
	if err != nil {
		return fmt.Errorf("statting NickelMenu: %w\n", err)
	}
	bar := progressbar.DefaultBytes(nmStat.Size(), "unpacking NickelMenu")
	if err := Extract(ctx, Tgz, io.TeeReader(nmArchive, bar), combined); err != nil {
		return fmt.Errorf("NickelMenu unpack: %w\n", err)
	}

	fwFile, err := download(updateUrl, "fw-*.zip", "firmware")
	if err != nil {
		return fmt.Errorf("downloading firmware zip: %w\n", err)
	}
	defer fwFile.Close()
	defer os.Remove(fwFile.Name())

	kRoot, err := os.CreateTemp("", "KoboRoot-*.tgz")
	if err != nil {
		return fmt.Errorf("creating temp firmware root: %w\n", err)
	}
	defer kRoot.Close()
	defer os.Remove(kRoot.Name())

	fwStat, err := fwFile.Stat()
	if err != nil {
		return fmt.Errorf("statting firmware: %w\n", err)
	}
	fwBar := progressbar.DefaultBytes(fwStat.Size(), "extracting firmware zip")
	if err := ExtractPrefix(ctx, Zip, &progressFile{File: fwFile, bar: fwBar}, Prefixes{
		"upgrade/":        filepath.Join(root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    kRoot.Name(),
	}); err != nil {
		return fmt.Errorf("extracting firmware: %w\n", err)
	}

	if _, err := kRoot.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking firmware root: %w\n", err)
	}
	krStat, err := kRoot.Stat()
	if err != nil {
		return fmt.Errorf("statting firmware root: %w\n", err)
	}
	krBar := progressbar.DefaultBytes(krStat.Size(), "unpacking firmware root")
	if err := Extract(ctx, Tgz, io.TeeReader(kRoot, krBar), combined); err != nil {
		return fmt.Errorf("firmware root unpack: %w\n", err)
	}

	res, err := os.Create(filepath.Join(root, ".kobo", "KoboRoot.tgz"))
	if err != nil {
		return fmt.Errorf("creating combined root: %w\n", err)
	}
	defer res.Close()
	files, err := filesFromDir(ctx, combined)
	if err != nil {
		return fmt.Errorf("gathering firmware root files: %w\n", err)
	}

	var n int
	var packBar *progressbar.ProgressBar
	for i, fi := range files {
		if !fi.IsDir() {
			n++
			origOpen := fi.Open
			fi.Open = func() (fs.File, error) {
				packBar.Add(1)
				return origOpen()
			}
			files[i] = fi
		}
	}
	packBar = progressbar.Default(int64(n), "packing combined root")
	if err := Tgz.Archive(ctx, res, files); err != nil {
		return fmt.Errorf("pack combined: %w\n", err)
	}

	if err := res.Close(); err != nil {
		return fmt.Errorf("finalizing combined root: %w\n", err)
	}

	fmt.Println("Done.")
	return nil
}
