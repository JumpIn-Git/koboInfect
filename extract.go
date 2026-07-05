package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
)

var TgzFormat = archives.CompressedArchive{
	Compression: archives.Gz{},
	Extraction:  archives.Tar{},
}
var ZipFormat = archives.Zip{}

type Prefixes map[string]string // [prefix]folder

func extractFormat(ctx context.Context, format archives.Extractor, src string, getOut func(f archives.FileInfo) string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	err = format.Extract(ctx, file, func(ctx context.Context, f archives.FileInfo) error {
		out := getOut(f)
		if out == "" {
			return nil
		}
		if f.IsDir() {
			return os.MkdirAll(out, f.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}

		stream, err := f.Open()
		if err != nil {
			return err
		}
		defer stream.Close()

		outFile, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, stream)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	return nil
}

func Extract(ctx context.Context, format archives.Extractor, src, dst string) error {
	return extractFormat(ctx, format, src, func(f archives.FileInfo) string {
		// No zip slip check for simplicity (trusted sources only)
		return filepath.Join(dst, f.NameInArchive)
	})
}

func ExtractPrefix(ctx context.Context, format archives.Extractor, src string, prefixes Prefixes) error {
	return extractFormat(ctx, format, src, func(f archives.FileInfo) string {
		var out string
		for prefix, folder := range prefixes {
			if !strings.HasPrefix(f.NameInArchive, prefix) {
				continue
			}
			if strings.HasSuffix(prefix, "/") {
				out = filepath.Join(folder, strings.TrimPrefix(f.NameInArchive, prefix))
			} else if f.NameInArchive == prefix {
				out = folder
			}
			break
		}
		return out
	})
}
