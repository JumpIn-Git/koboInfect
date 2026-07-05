package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pgaskin/koboutils/v2/kobo"
	"github.com/woozymasta/tgz"
)

var Root string
var Ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)

func main() {
	var nmConfigPath string
	var nmFile os.FileInfo

	flag.StringVar(&Root, "kobo", "", "Path to the Kobo root")
	flag.StringVar(&nmConfigPath, "nm-config", "", "Path to a NickelMenu config ")
	flag.Parse()

	if nmConfigPath != "" {
		var err error
		if nmFile, err = os.Stat(nmConfigPath); err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("NickelMenu config file does not exist: %s", nmConfigPath)
			}
			log.Fatalf("failed to check NickelMenu config file: %v", err)
		}
	}

	if Root == "" {
		var err error
		Root, err = GetKobo()
		if err != nil {
			log.Fatalf("failed to find any kobo's: %v", err)
		}
	} else {
		if !kobo.IsKobo(Root) {
			log.Fatalf("specified kobo path isn't a kobo")
		}
	}

	if err := GetPlato(); err != nil {
		log.Fatalf("plato: %v", err)
	}
	if err := GetKoreader(); err != nil {
		log.Fatalf("koreader: %v", err)
	}

	fmt.Println("checking for fw update...")
	updateUrl, err := UpgradeCheck()
	if err != nil {
		log.Fatalf("fw: %v", err)
	}

	upgrading := updateUrl != ""

	nmTmp, err := SaveNm(upgrading)
	if err != nil {
		log.Fatalf("nickelmenu: %v", err)
	}
	if nmTmp != "" {
		defer os.Remove(nmTmp)
	}

	if nmConfigPath != "" {
		if err := copyFile(nmConfigPath, filepath.Join(Root, ".adds", "nm", nmFile.Name())); err != nil {
			log.Fatalf("couldn't copy nm config: %v", err)
		}
	}

	if !upgrading {
		fmt.Println("Done.")
		return
	}

	combined, err := os.MkdirTemp("", "combined*") // nm + kobo-update root
	if err != nil {
		log.Fatalf("failed to create temp directory for merging: %v", err)
	}
	defer os.RemoveAll(combined)

	fmt.Println("unpacking NickelMenu...")
	if err := Extract(Ctx, TgzFormat, nmTmp, combined); err != nil {
		log.Fatalf("nickelmenu unpack: %v", err)
	}

	fmt.Println("downloading fw...")
	fwTmp, err := saveArchive(updateUrl, "fw-*.zip", true) // kobo-update.zip
	defer os.Remove(fwTmp)
	kRoot, err := os.CreateTemp("", "KoboRoot-*.tgz")
	if err != nil {
		log.Fatalf("failed to create temp file for fw root: %v", err)
	}
	defer kRoot.Close()

	fmt.Println("extracting fw...")
	if err := ExtractPrefix(Ctx, ZipFormat, fwTmp, Prefixes{
		"upgrade/":        filepath.Join(Root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(Root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    kRoot.Name(),
	}); err != nil {
		log.Fatalf("failed to extract fw: %v", err)
	}

	fmt.Println("extractng fw root...")
	if err := Extract(Ctx, TgzFormat, kRoot.Name(), combined); err != nil {
		log.Fatalf("fw unpack: %v", err)
	}

	target := filepath.Join(Root, ".kobo", "KoboRoot.tgz")
	fmt.Printf("packing combined archive to %s...\n", target)
	if err := tgz.Pack(combined, target); err != nil {
		log.Fatalf("pack combined: %v", err)
	}

	fmt.Println("Done.")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(dst), err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}
	return nil
}

func GetKobo() (string, error) {
	kobos, err := kobo.Find()
	if err != nil {
		return "", err
	}

	var root string
	if len(kobos) < 1 {
		return "", errors.New("No kobo's found, are any mounted?")
	} else if len(kobos) == 1 {
		root = kobos[0]
	} else {
		prompt := promptui.Select{Label: "Select a kobo", Items: kobos}
		_, Root, err = prompt.Run()
		print(Root)
		if err != nil {
			return "", err
		}
		os.Exit(0)
	}
	return root, nil
}
