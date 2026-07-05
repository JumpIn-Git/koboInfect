package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTGZ unpacks a .tar.gz / .tgz archive to the destination directory.
// It ensures that directories are created with write permissions (0755) to prevent
// permission issues when extracting files into directories that the archive specifies as read-only.
func ExtractTGZ(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to initialize gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Sanitize and construct the destination path
		cleanedName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanedName, "..") || strings.HasPrefix(cleanedName, "/") {
			return fmt.Errorf("suspicious file path in archive: %s", header.Name)
		}

		targetPath := filepath.Join(destDir, cleanedName)

		switch header.Typeflag {
		case tar.TypeDir:
			// Enforce owner write permissions (mode | 0200) so we can write child files/folders
			mode := header.FileInfo().Mode().Perm() | 0200
			if err := os.MkdirAll(targetPath, mode); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Ensure parent directory exists with owner write permissions
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			// Create the file
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file content for %s: %w", targetPath, err)
			}
			outFile.Close()

		case tar.TypeSymlink:
			// Ensure parent directory exists
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			os.Remove(targetPath) // remove existing file/symlink
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", targetPath, header.Linkname, err)
			}

		case tar.TypeLink: // hard links
			// Ensure parent directory exists
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			os.Remove(targetPath)
			oldPath := filepath.Join(destDir, header.Linkname)
			if err := os.Link(oldPath, targetPath); err != nil {
				return fmt.Errorf("failed to create hard link %s -> %s: %w", targetPath, oldPath, err)
			}
		}
	}
	return nil
}
