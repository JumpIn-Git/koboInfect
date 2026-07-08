package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/pgaskin/koboutils/v2/kobo"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/ini.v1"
)

var Root string

func main() {
	ctx := context.Background()
	os.Exit(run(ctx))
}

func run(ctx context.Context) int {
	var nmConfigPath string
	var sideloadMode bool

	flag.StringVar(&Root, "kobo", "", "Path to the Kobo root")
	flag.StringVar(&nmConfigPath, "nm-config", "", "Path to a NickelMenu config to copy")
	flag.BoolVar(&sideloadMode, "sideloadMode", false, "Enable sideload mode, no account needed (use after factory reset)")
	flag.Parse()

	if Root == "" {
		var err error
		Root, err = GetKobo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "finding Kobos: %v\n", err)
			return 1
		}
	} else {
		if !kobo.IsKobo(Root) {
			fmt.Fprintf(os.Stderr, "%s doesn't seem to be a Kobo root\n", Root)
			return 1
		}
	}

	if nmConfigPath != "" {
		if nmFile, err := os.Stat(nmConfigPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "NickelMenu config file does not exist: %s\n", nmConfigPath)
				return 1
			}
			fmt.Fprintf(os.Stderr, "checking NickelMenu config: %v\n", err)
			return 1
		} else if err := copyFile(nmConfigPath, filepath.Join(Root, ".adds", "nm", nmFile.Name())); err != nil {
			fmt.Fprintf(os.Stderr, "copying NickelMenu config: %v\n", err)
			return 1
		}
	}
	cfgPath := filepath.Join(Root, ".kobo", "Kobo", "Kobo eReader.conf")
	cfgBak := filepath.Join(Root, ".kobo", "Kobo", "Kobo eReader.conf.bak")
	if err := copyFile(cfgPath, cfgBak); err != nil {
		fmt.Fprintf(os.Stderr, "backing up Kobo eReader.conf: %v\n", err)
		return 1
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{SpaceBeforeInlineComment: true}, cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening Kobo eReader.conf: %v\n", err)
		return 1
	}
	cfg.Section("FeatureSettings").Key("ExcludeSyncFolders").SetValue(`(\\.(?!kobo|adobe).+|([^.][^/]*/)+\\..+)`)
	if sideloadMode {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "saving Kobo eReader.conf: %v\n", err)
		return 1
	}

	var install []string
	if err := huh.NewMultiSelect[string]().
		Title("What to install? (space to toggle, enter to confirm)").
		Options(huh.NewOptions([]string{"Plato", "KOReader"}...)...).
		Value(&install).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "selection: %v\n", err)
		return 1
	}
	for _, s := range install {
		switch s {
		case "Plato":
			if err := GetPlato(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "plato: %v\n", err)
				return 1
			}
		case "KOReader":
			if err := GetKoreader(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "koreader: %v\n", err)
				return 1
			}
		}
	}

	fmt.Println("checking for firmware upgrade...")
	updateUrl, err := UpgradeCheck()
	if err != nil {
		fmt.Fprintf(os.Stderr, "firmware check: %v\n", err)
		return 1
	}

	upgrading := updateUrl != ""

	nmArchive, err := GetNM(ctx, upgrading)
	if err != nil {
		fmt.Fprintf(os.Stderr, "saving NickelMenu: %v\n", err)
		return 1
	}
	if nmArchive != nil {
		defer nmArchive.Close()
		defer os.Remove(nmArchive.Name())
	}

	if !upgrading {
		fmt.Println("Done.")
		return 0
	}

	combined, err := os.MkdirTemp("", "combined*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating combined firmware root dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(combined)

	fmt.Println("unpacking NickelMenu...")
	nmStat, err := nmArchive.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "statting NickelMenu: %v\n", err)
		return 1
	}
	bar := progressbar.DefaultBytes(nmStat.Size(), "unpacking NickelMenu")
	if err := Extract(ctx, Tgz, io.TeeReader(nmArchive, bar), combined); err != nil {
		fmt.Fprintf(os.Stderr, "NickelMenu unpack: %v\n", err)
		return 1
	}

	fmt.Println("downloading firmware...")
	fwFile, err := download(updateUrl, "fw-*.zip", "firmware")
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloading firmware: %v\n", err)
		return 1
	}
	defer fwFile.Close()
	defer os.Remove(fwFile.Name())

	kRoot, err := os.CreateTemp("", "KoboRoot-*.tgz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating temp firmware root: %v\n", err)
		return 1
	}
	defer kRoot.Close()
	defer os.Remove(kRoot.Name())

	fmt.Println("extracting fw...")
	fwStat, err := fwFile.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "statting firmware: %v\n", err)
		return 1
	}
	fwBar := progressbar.DefaultBytes(fwStat.Size(), "extracting firmware zip")
	if err := ExtractPrefix(ctx, Zip, &progressFile{File: fwFile, bar: fwBar}, Prefixes{
		"upgrade/":        filepath.Join(Root, ".kobo", "upgrade"),
		"manifest.md5sum": filepath.Join(Root, ".kobo", "manifest.md5sum"),
		"KoboRoot.tgz":    kRoot.Name(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "extracting firmware: %v\n", err)
		return 1
	}

	fmt.Println("extracting firmware root...")
	if _, err := kRoot.Seek(0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "seeking firmware root: %v\n", err)
		return 1
	}
	krStat, err := kRoot.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "statting firmware root: %v\n", err)
		return 1
	}
	krBar := progressbar.DefaultBytes(krStat.Size(), "unpacking firmware root")
	if err := Extract(ctx, Tgz, io.TeeReader(kRoot, krBar), combined); err != nil {
		fmt.Fprintf(os.Stderr, "firmware root unpack: %v\n", err)
		return 1
	}

	target := filepath.Join(Root, ".kobo", "KoboRoot.tgz")
	fmt.Printf("packing combined root to %s...\n", target)
	f, err := os.Create(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating combined root: %v\n", err)
		return 1
	}
	defer f.Close()
	files, err := filesFromDir(ctx, combined)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gathering firmware root files: %v\n", err)
		return 1
	}
	packBar := progressbar.DefaultBytes(-1, "packing combined root")
	if err := Tgz.Archive(ctx, io.MultiWriter(f, packBar), files); err != nil {
		fmt.Fprintf(os.Stderr, "pack combined: %v\n", err)
		return 1
	}

	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "finalizing combined root: %v\n", err)
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
	return out.Close()
}

func GetKobo() (string, error) {
	kobos, err := kobo.Find()
	if err != nil {
		return "", err
	}

	var root string
	if len(kobos) < 1 {
		return "", errors.New("no Kobos found, are any mounted?")
	} else if len(kobos) == 1 {
		root = kobos[0]
	} else {
		if err := huh.NewSelect[string]().
			Title("Select a kobo").
			Options(huh.NewOptions(kobos...)...).
			Value(&root).Run(); err != nil {
			return "", err
		}
	}
	return root, nil
}
