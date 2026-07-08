package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/schollz/progressbar/v3"
)

var Tgz = archives.CompressedArchive{
	Compression: archives.Gz{},
	Extraction:  archives.Tar{},
	Archival:    archives.Tar{},
}
var Zip = archives.Zip{}

type Prefixes map[string]string

func extractFormat(ctx context.Context, format archives.Extractor, r io.Reader, getOut func(f archives.FileInfo) string) error {
	err := format.Extract(ctx, r, func(ctx context.Context, f archives.FileInfo) error {
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
		return fmt.Errorf("extracting archive: %w\n", err)
	}
	return nil
}

func Extract(ctx context.Context, format archives.Extractor, r io.Reader, dst string) error {
	return extractFormat(ctx, format, r, func(f archives.FileInfo) string {
		return filepath.Join(dst, f.NameInArchive)
	})
}

func ExtractPrefix(ctx context.Context, format archives.Extractor, r io.Reader, prefixes Prefixes) error {
	return extractFormat(ctx, format, r, func(f archives.FileInfo) string {
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

func filesFromDir(ctx context.Context, dir string) ([]archives.FileInfo, error) {
	return archives.FilesFromDisk(ctx, nil, map[string]string{
		filepath.Clean(dir) + string(filepath.Separator): ".",
	})
}

type progressFile struct {
	*os.File
	bar *progressbar.ProgressBar
}

func (pf *progressFile) Read(p []byte) (int, error) {
	n, err := pf.File.Read(p)
	if n > 0 {
		_ = pf.bar.Add(n)
	}
	return n, err
}

func (pf *progressFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := pf.File.ReadAt(p, off)
	if n > 0 {
		_ = pf.bar.Add(n)
	}
	return n, err
}


