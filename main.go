package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/pgaskin/koboutils/v2/kobo"
	"github.com/woozymasta/tgz"
)

var Root string

func main() {
	// Root, err := GetKobo()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	p, err := os.Getwd()
	Root = filepath.Join(p, "tmp")

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

	if updateUrl == "" {
		_, err := SaveNm(false)
		if err != nil {
			log.Fatalf("nickelmenu: %v", err)
		}
		return
	}

	nmTmp, err := SaveNm(true)
	if err != nil {
		log.Fatalf("nickelmenu: %v", err)
	}

	fwTmp, err := os.CreateTemp("", "fw-*.tgz")
	if err != nil {
		log.Fatalf("fw KoboRoot: %v", err)
	}
	fmt.Println("extracting fw")
	if err := ExtractZipPrefixes(updateUrl, map[string]string{
		"upgrade/":        filepath.Join(Root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(Root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    fwTmp.Name(),
	}); err != nil {
		log.Fatalf("fw: %v", err)
	}
	fwTmp.Close()

	combined, err := os.MkdirTemp("", "combined*")
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := unpackTGZ(nmTmp, combined); err != nil {
		log.Fatalf("nickelmenu: %v", err)
	}
	if err := unpackTGZ(fwTmp.Name(), combined); err != nil {
		log.Fatalf("fw: %v", err)
	}

	if err := tgz.Pack(combined, filepath.Join(Root, ".kobo", "KoboRoot.tgz")); err != nil {
		log.Fatalf("pack: %v", err)
	}
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
