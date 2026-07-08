package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

func optWrite(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
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

func editConf(root string, sideload bool) error {
	cfgPath := filepath.Join(root, ".kobo", "Kobo", "Kobo eReader.conf")
	cfgBak := filepath.Join(root, ".kobo", "Kobo", "Kobo eReader.conf.bak")
	if err := copyFile(cfgPath, cfgBak); err != nil {
		return fmt.Errorf("backing up Kobo eReader.conf: %w", err)
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{SpaceBeforeInlineComment: true}, cfgPath)
	if err != nil {
		return fmt.Errorf("opening Kobo eReader.conf: %w", err)
	}
	cfg.Section("FeatureSettings").Key("ExcludeSyncFolders").SetValue(`(\\.(?!kobo|adobe).+|([^.][^/]*/)+\\..+)`)
	if sideload {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		return fmt.Errorf("saving Kobo eReader.conf: %w", err)
	}
	return nil
}

func copyNm(root string, path string) error {
	if path != "" {
		if nmFile, err := os.Stat(path); err != nil {
			return fmt.Errorf("checking NickelMenu config: %w", err)
		} else if err := copyFile(path, filepath.Join(root, ".adds", "nm", nmFile.Name())); err != nil {
			return fmt.Errorf("copying NickelMenu config: %w", err)
		}
	}
	return nil
}
