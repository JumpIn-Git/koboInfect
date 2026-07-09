package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/pgaskin/koboutils/v2/kobo"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	nmConfigPath := flag.String("nm-config", "", "Path to a Nickelmenu config to copy.")
	sideloadMode := flag.Bool("sideloadMode", false, "Enable sideload mode, no account needed (use after factory reset)")
	var root string
	flag.StringVar(&root, "kobo", "", "Path to the Kobo root")
	flag.Parse()

	if root == "" {
		var err error
		root, err = GetKobo()
		if err != nil {
			return fmt.Errorf("finding Kobos: %w\n", err)
		}
	} else {
		if !kobo.IsKobo(root) {
			return fmt.Errorf("%s doesn't seem to be a Kobo root\n", root)
		}
	}

	if err := os.MkdirAll(filepath.Join(root, ".adds", "nm"), 0755); err != nil {
		return fmt.Errorf("failed to make .adds/nm: %w\n", err)
	}
	if err := copyNm(root, *nmConfigPath); err != nil {
		return err
	}
	if err := editConf(root, *sideloadMode); err != nil {
		return err
	}

	install, err := selectAddons()
	if err != nil {
		return fmt.Errorf("selection: %w\n", err)
	}
	if slices.Contains(install, "Plato") {
		if err := GetPlato(ctx, root); err != nil {
			return fmt.Errorf("plato: %w\n", err)
		}
	}
	if slices.Contains(install, "KOReader") {
		if err := GetKoreader(ctx, root); err != nil {
			return fmt.Errorf("koreader: %w\n", err)
		}
	}

	return handleFirmware(ctx, root)
}

func GetKobo() (root string, err error) {
	kobos, err := kobo.Find()
	if err != nil {
		return "", err
	}

	if len(kobos) < 1 {
		return "", errors.New("no Kobos found, are any mounted?")
	} else if len(kobos) == 1 {
		root = kobos[0]
		fmt.Printf("Found %s!\n", root)
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

func selectAddons() (install []string, err error) {
	err = huh.NewMultiSelect[string]().
		Title("What to install? (space to toggle)").
		Options(huh.NewOptions("Plato", "KOReader")...).
		Value(&install).Run()
	return install, err
}
