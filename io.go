package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

func optWrite(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w\n", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w\n", dst, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("copying content: %w\n", err)
	}
	return out.Close()
}

func editConf(root string, sideload bool) error {
	cfgPath := filepath.Join(root, ".kobo", "Kobo", "Kobo eReader.conf")
	cfgBak := filepath.Join(root, ".kobo", "Kobo", "Kobo eReader.conf.bak")
	if err := copyFile(cfgPath, cfgBak); err != nil {
		return fmt.Errorf("Kobo eReader.conf.bak: %w\n", err)
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{SpaceBeforeInlineComment: true}, cfgPath)
	if err != nil {
		return fmt.Errorf("Kobo eReader.conf: %w\n", err)
	}
	cfg.Section("FeatureSettings").Key("ExcludeSyncFolders").SetValue(`(\\.(?!kobo|adobe).+|([^.][^/]*/)+\\..+)`)
	if sideload {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		return fmt.Errorf("saving Kobo eReader.conf: %w\n", err)
	}
	return nil
}

func copyNm(root string, path string) error {
	if path != "" {
		if nmFile, err := os.Stat(path); err != nil {
			return fmt.Errorf("NickelMenu config: %w\n", err)
		} else if err := copyFile(path, filepath.Join(root, ".adds", "nm", nmFile.Name())); err != nil {
			return fmt.Errorf("NickelMenu config: %w\n", err)
		}
	}
	return nil
}
