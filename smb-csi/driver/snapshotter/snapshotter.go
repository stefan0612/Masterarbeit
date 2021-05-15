package snapshotter

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const ext = ".snap"

func CreateSnapshot(volumePath string, outPath string) error {

	outPath = outPath + ext

	if _, err := os.Stat(volumePath); err != nil {
		return err
	}

	snapFile, err := os.Create(outPath)
	if err != nil { return err }

	var fileWriter io.WriteCloser = snapFile
	gzw := gzip.NewWriter(fileWriter)
	tw := tar.NewWriter(gzw)

	return filepath.Walk(volumePath, func(file string, fi os.FileInfo, err error) error {
		if err != nil { return err }
		if !fi.Mode().IsRegular() { return nil }

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil { return err }
		header.Name = strings.TrimPrefix(strings.Replace(file, volumePath, "", -1), string(filepath.Separator))
		header.Size = fi.Size()
		header.Mode = int64(fi.Mode())
		header.ModTime = fi.ModTime()
		if err := tw.WriteHeader(header); err != nil { return err }

		f, err := os.Open(file)
		if err != nil { return err }

		if _, err := io.Copy(tw, f); err != nil { return err }
		if err := f.Close(); err != nil { return err }

		return nil
	})
}

func DeleteSnapshot(snapPath string) error {
	return os.RemoveAll(snapPath + ext)
}
