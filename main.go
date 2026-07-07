package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	os.Exit(run())
}

func run() int {
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
				fmt.Fprintf(os.Stderr, "NickelMenu config file does not exist: %s\n", nmConfigPath)
				return 1
			}
			fmt.Fprintf(os.Stderr, "checking NickelMenu config: %v\n", err)
			return 1
		}
	}

	if Root == "" {
		var err error
		Root, err = GetKobo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "finding Kobos: %v\n", err)
			return 1
		}
	} else {
		if !kobo.IsKobo(Root) {
			fmt.Fprintln(os.Stderr, "not a Kobo root")
			return 1
		}
	}

	if err := GetPlato(); err != nil {
		fmt.Fprintf(os.Stderr, "plato: %v\n", err)
		return 1
	}
	if err := GetKoreader(); err != nil {
		fmt.Fprintf(os.Stderr, "koreader: %v\n", err)
		return 1
	}

	fmt.Println("checking for fw update...")
	updateUrl, err := UpgradeCheck()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fw: %v\n", err)
		return 1
	}

	upgrading := updateUrl != ""

	nmArchive, err := SaveNm(upgrading)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nickelmenu: %v\n", err)
		return 1
	}
	if nmArchive != nil {
		defer nmArchive.Close()
		defer os.Remove(nmArchive.Name())
	}

	if nmConfigPath != "" {
		if err := copyFile(nmConfigPath, filepath.Join(Root, ".adds", "nm", nmFile.Name())); err != nil {
			fmt.Fprintf(os.Stderr, "copying nm config: %v\n", err)
			return 1
		}
	}
	cfgPath := filepath.Join(Root, ".kobo", "Kobo", "Kobo eReader.conf")
	cfgBak := filepath.Join(Root, ".kobo", "Kobo", "Kobo eReader.conf.bak")
	if err := copyFile(cfgPath, cfgBak); err != nil {
		fmt.Fprintf(os.Stderr, "backing up .conf: %v\n", err)
		return 1
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{SpaceBeforeInlineComment: true}, cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening .conf: %v\n", err)
		return 1
	}
	cfg.Section("FeatureSettings").Key("ExcludeSyncFolders").SetValue(`(\\.(?!kobo|adobe).+|([^.][^/]*/)+\\..+)`)
	if sideloadMode {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "saving .conf: %v\n", err)
		return 1
	}

	if !upgrading {
		fmt.Println("Done.")
		return 0
	}

	combined, err := os.MkdirTemp("", "combined*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating temp directory: %v\n", err)
		return 1
	}
	defer os.RemoveAll(combined)

	fmt.Println("unpacking NickelMenu...")
	if err := Extract(Ctx, Tgz, nmArchive, combined); err != nil {
		fmt.Fprintf(os.Stderr, "nickelmenu unpack: %v\n", err)
		return 1
	}

	fmt.Println("downloading fw...")
	fwFile, err := download(updateUrl, "fw-*.zip")
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloading firmware: %v\n", err)
		return 1
	}
	defer fwFile.Close()
	defer os.Remove(fwFile.Name())

	kRoot, err := os.CreateTemp("", "KoboRoot-*.tgz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating temp fw root: %v\n", err)
		return 1
	}
	defer kRoot.Close()
	defer os.Remove(kRoot.Name())

	fmt.Println("extracting fw...")
	if err := ExtractPrefix(Ctx, Zip, fwFile, Prefixes{
		"upgrade/":        filepath.Join(Root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(Root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    kRoot.Name(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "extracting fw: %v\n", err)
		return 1
	}

	fmt.Println("extracting fw root...")
	kr, err := os.Open(kRoot.Name())
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening fw root: %v\n", err)
		return 1
	}
	defer kr.Close()
	if err := Extract(Ctx, Tgz, kr, combined); err != nil {
		fmt.Fprintf(os.Stderr, "fw unpack: %v\n", err)
		return 1
	}

	target := filepath.Join(Root, ".kobo", "KoboRoot.tgz")
	fmt.Printf("packing combined archive to %s...\n", target)
	f, err := os.Create(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating combined archive: %v\n", err)
		return 1
	}
	defer f.Close()
	files, err := archives.FilesFromDisk(Ctx, nil, map[string]string{
		combined + string(filepath.Separator): ".",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gathering files: %v\n", err)
		return 1
	}
	if err := Tgz.Archive(Ctx, f, files); err != nil {
		fmt.Fprintf(os.Stderr, "pack combined: %v\n", err)
		return 1
	}

	fmt.Println("Done.")
	return 0
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
