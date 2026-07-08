package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

func handleFirmware(ctx context.Context, root string) error {
	fmt.Println("checking for firmware upgrade...")
	updateUrl, err := UpgradeCheck(root)
	if err != nil {
		return fmt.Errorf("firmware check: %w", err)
	}

	upgrading := updateUrl != ""

	nmArchive, err := GetNM(ctx, upgrading, root)
	if err != nil {
		return fmt.Errorf("saving NickelMenu: %w", err)
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
		return fmt.Errorf("creating combined firmware root dir: %w", err)
	}
	defer os.RemoveAll(combined)

	nmStat, err := nmArchive.Stat()
	if err != nil {
		return fmt.Errorf("statting NickelMenu: %w", err)
	}
	bar := progressbar.DefaultBytes(nmStat.Size(), "unpacking NickelMenu")
	if err := Extract(ctx, Tgz, io.TeeReader(nmArchive, bar), combined); err != nil {
		return fmt.Errorf("NickelMenu unpack: %w", err)
	}

	fwFile, err := download(updateUrl, "fw-*.zip", "firmware")
	if err != nil {
		return fmt.Errorf("downloading firmware zip: %w", err)
	}
	defer fwFile.Close()
	defer os.Remove(fwFile.Name())

	kRoot, err := os.CreateTemp("", "KoboRoot-*.tgz")
	if err != nil {
		return fmt.Errorf("creating temp firmware root: %w", err)
	}
	defer kRoot.Close()
	defer os.Remove(kRoot.Name())

	fwStat, err := fwFile.Stat()
	if err != nil {
		return fmt.Errorf("statting firmware: %w", err)
	}
	fwBar := progressbar.DefaultBytes(fwStat.Size(), "extracting firmware zip")
	if err := ExtractPrefix(ctx, Zip, &progressFile{File: fwFile, bar: fwBar}, Prefixes{
		"upgrade/":        filepath.Join(root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    kRoot.Name(),
	}); err != nil {
		return fmt.Errorf("extracting firmware: %w", err)
	}

	if _, err := kRoot.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking firmware root: %w", err)
	}
	krStat, err := kRoot.Stat()
	if err != nil {
		return fmt.Errorf("statting firmware root: %w", err)
	}
	krBar := progressbar.DefaultBytes(krStat.Size(), "unpacking firmware root")
	if err := Extract(ctx, Tgz, io.TeeReader(kRoot, krBar), combined); err != nil {
		return fmt.Errorf("firmware root unpack: %w", err)
	}

	f, err := os.Create(filepath.Join(root, ".kobo", "KoboRoot.tgz"))
	if err != nil {
		return fmt.Errorf("creating combined root: %w", err)
	}
	defer f.Close()
	files, err := filesFromDir(ctx, combined)
	if err != nil {
		return fmt.Errorf("gathering firmware root files: %w", err)
	}
	packBar := progressbar.DefaultBytes(-1, "packing combined root")
	if err := Tgz.Archive(ctx, io.MultiWriter(f, packBar), files); err != nil {
		return fmt.Errorf("pack combined: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("finalizing combined root: %w", err)
	}

	fmt.Println("Done.")
	return nil
}
