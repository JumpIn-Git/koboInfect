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

	"github.com/manifoldco/promptui"
	"github.com/mholt/archives"
	"github.com/pgaskin/koboutils/v2/kobo"
	"gopkg.in/ini.v1"
)

var Root string
var Ctx = context.Background()

func main() {
	var nmConfigPath string
	var nmFile os.FileInfo
	var sideloadMode bool

	flag.StringVar(&Root, "kobo", "", "Path to the Kobo root")
	flag.StringVar(&nmConfigPath, "nm-config", "", "Path to a NickelMenu config to copy")
	flag.BoolVar(&sideloadMode, "sideloadMode", false, "Enable sideload mode, no account needed (use after factory reset)")
	flag.Parse()

	if nmConfigPath != "" {
		var err error
		if nmFile, err = os.Stat(nmConfigPath); err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("NickelMenu config file does not exist: %s", nmConfigPath)
			}
			log.Fatalf("checking NickelMenu config: %v", err)
		}
	}

	if Root == "" {
		var err error
		Root, err = GetKobo()
		if err != nil {
			log.Fatalf("finding Kobos: %v", err)
		}
	} else {
		if !kobo.IsKobo(Root) {
			log.Fatalf("not a Kobo root")
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

	nmArchive, err := SaveNm(upgrading)
	if err != nil {
		log.Fatalf("nickelmenu: %v", err)
	}
	if nmArchive != nil {
		defer nmArchive.Close()
		defer os.Remove(nmArchive.Name())
	}

	if nmConfigPath != "" {
		if err := copyFile(nmConfigPath, filepath.Join(Root, ".adds", "nm", nmFile.Name())); err != nil {
			log.Fatalf("copying nm config: %v", err)
		}
	}
	cfgPath := filepath.Join(Root, ".kobo", "Kobo", "Kobo eReader.conf")
	cfgBak := filepath.Join(Root, ".kobo", "Kobo", "Kobo eReader.conf.bak")
	if err := copyFile(cfgPath, cfgBak); err != nil {
		log.Fatalf("failed to backup .conf: %v", err)
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{SpaceBeforeInlineComment: true}, cfgPath)
	if err != nil {
		log.Fatalf("failed to open .conf: %v", err)
	}
	cfg.Section("FeatureSettings").Key("ExcludeSyncFolders").SetValue(`(\\.(?!kobo|adobe).+|([^.][^/]*/)+\\..+)`)
	if sideloadMode {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		log.Fatalf("failed to save .conf: %v", err)
	}

	if !upgrading {
		fmt.Println("Done.")
		return
	}

	combined, err := os.MkdirTemp("", "combined*")
	if err != nil {
		log.Fatalf("creating temp directory: %v", err)
	}
	defer os.RemoveAll(combined)

	fmt.Println("unpacking NickelMenu...")
	if err := Extract(Ctx, TgzFormat, nmArchive, combined); err != nil {
		log.Fatalf("nickelmenu unpack: %v", err)
	}

	fmt.Println("downloading fw...")
	fwFile, err := download(updateUrl, "fw-*.zip")
	if err != nil {
		log.Fatalf("downloading firmware: %v", err)
	}
	defer fwFile.Close()
	defer os.Remove(fwFile.Name())

	kRoot, err := os.CreateTemp("", "KoboRoot-*.tgz")
	if err != nil {
		log.Fatalf("creating temp fw root: %v", err)
	}
	defer kRoot.Close()
	defer os.Remove(kRoot.Name())

	fmt.Println("extracting fw...")
	if err := ExtractPrefix(Ctx, ZipFormat, fwFile, Prefixes{
		"upgrade/":        filepath.Join(Root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(Root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    kRoot.Name(),
	}); err != nil {
		log.Fatalf("extracting fw: %v", err)
	}

	fmt.Println("extracting fw root...")
	kr, err := os.Open(kRoot.Name())
	if err != nil {
		log.Fatalf("opening fw root: %v", err)
	}
	defer kr.Close()
	if err := Extract(Ctx, TgzFormat, kr, combined); err != nil {
		log.Fatalf("fw unpack: %v", err)
	}

	target := filepath.Join(Root, ".kobo", "KoboRoot.tgz")
	fmt.Printf("packing combined archive to %s...\n", target)
	f, err := os.Create(target)
	if err != nil {
		log.Fatalf("creating combined archive: %v", err)
	}
	defer f.Close()
	files, err := archives.FilesFromDisk(Ctx, nil, map[string]string{
		combined + string(filepath.Separator): ".",
	})
	if err != nil {
		log.Fatalf("gathering files: %v", err)
	}
	if err := TgzFormat.Archive(Ctx, f, files); err != nil {
		log.Fatalf("pack combined: %v", err)
	}

	fmt.Println("Done.")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", filepath.Dir(dst), err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("copying content: %w", err)
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
		return "", errors.New("no Kobos found")
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
