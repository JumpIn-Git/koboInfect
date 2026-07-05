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

	"charm.land/huh/v2"
	"github.com/mholt/archives"
	"github.com/pgaskin/koboutils/v2/kobo"
	"github.com/woozymasta/tgz"
)

var Root string

func main() {
	path, err := SaveNm(true)
	if err != nil {
		panic(err)
	}
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	format := archives.CompressedArchive{
		Compression: archives.Gz{},
		Extraction:  archives.Tar{},
	}
	ctx := context.Background()
	destinationDir := "./my-extracted-files"

	// 4. Run the extraction loop
	err = format.Extract(ctx, file, func(ctx context.Context, f archives.FileInfo) error {
		// Construct the destination path on your local disk
		outputPath := filepath.Join(destinationDir, f.NameInArchive)

		// Handle Directories
		if f.IsDir() {
			return os.MkdirAll(outputPath, f.Mode())
		}

		// Ensure the parent directory exists (in case the tarball lists files before parent dirs)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		// Open the file inside the archive stream
		archiveFile, err := f.Open()
		if err != nil {
			return err
		}
		defer archiveFile.Close()

		// Create the file on your local hard drive
		localFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer localFile.Close()

		// Stream the data straight from the archive to your disk
		_, err = io.Copy(localFile, archiveFile)
		return err
	})

	if err != nil {
		panic(err)
	}

	os.Exit(0)
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

	combined, err := os.MkdirTemp("", "combined*")
	if err != nil {
		log.Fatalf("failed to create temp directory for merging: %v", err)
	}
	defer os.RemoveAll(combined)

	fmt.Println("unpacking NickelMenu...")
	if err := ExtractTGZ(nmTmp, combined); err != nil {
		log.Fatalf("nickelmenu unpack: %v", err)
	}

	fwTmp, err := os.CreateTemp("", "fw-*.tgz")
	if err != nil {
		log.Fatalf("fw KoboRoot: %v", err)
	}
	defer os.Remove(fwTmp.Name())

	fmt.Println("extracting fw...")
	if err := ExtractZipPrefixes(updateUrl, map[string]string{
		"upgrade/":        filepath.Join(Root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(Root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    fwTmp.Name(),
	}); err != nil {
		fwTmp.Close()
		log.Fatalf("fw extraction: %v", err)
	}
	fwTmp.Close()

	fmt.Println("unpacking fw...")
	if err := ExtractTGZ(fwTmp.Name(), combined); err != nil {
		log.Fatalf("fw unpack: %v", err)
	}

	targetArchive := filepath.Join(Root, ".kobo", "KoboRoot.tgz")
	fmt.Printf("packing combined archive to %s...\n", targetArchive)
	if err := tgz.Pack(combined, targetArchive); err != nil {
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
		return "", errors.New("No kobo's found.")
	} else if len(kobos) == 1 {
		root = kobos[0]
	} else {
		err := huh.NewSelect[string]().
			Title("Select a kobo:").
			Options(huh.NewOptions(kobos...)...).
			Value(&root).Run()
		if err != nil {
			return "", err
		}
	}
	return root, nil
}
