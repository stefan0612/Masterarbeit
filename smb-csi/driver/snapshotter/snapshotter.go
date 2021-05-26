package snapshotter

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func CreateSnapshot(volumePath string, snapFileOut string) error {
	var buf bytes.Buffer
	if err := compress(volumePath, &buf); err != nil {
		return err
	}

	if err := ioutil.WriteFile(snapFileOut, buf.Bytes(), 0777); err != nil {
		return err
	}
	return nil
}

func DeleteSnapshot(snapFile string) error {
	return os.RemoveAll(snapFile)
}

func ExtractSnap(snapFileIn string, outPath string) error {
	if createDirErr := os.MkdirAll(outPath, os.ModeDir); createDirErr != nil {
		return status.Errorf(codes.Internal, "Failed creating mount directory: %s", createDirErr.Error())
	}
	content, readErr := ioutil.ReadFile(snapFileIn)
	if readErr != nil {
		return readErr
	}
	buf := bytes.NewReader(content)
	if err := decompress(buf, outPath); err != nil {
		return err
	}
	return nil
}

func compress(src string, buf io.Writer) error {
	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	// is file a folder?
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		// get header
		header, err := tar.FileInfoHeader(fi, src)
		if err != nil {
			return err
		}
		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// get content
		data, err := os.Open(src)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, data); err != nil {
			data.Close()
			return err
		}
		data.Close()
	} else if mode.IsDir() { // folder

		// walk through every file in the folder
		filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
			// generate tar header
			header, err := tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))
			fmt.Printf("header: %s", header.Name )

			// write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				if _, err := io.Copy(tw, data); err != nil {
					data.Close()
					return err
				}
				data.Close()
			}
			return nil
		})
	} else {
		return fmt.Errorf("error: file type not supported")
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}

func decompress(src io.Reader, dst string) error {
	// ungzip
	zr, err := gzip.NewReader(src)
	defer zr.Close()
	if err != nil {
		return err
	}
	// untar
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := header.Name

		target = filepath.Join(dst, header.Name)

		// check the type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it (with 0755 permission)
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it (with same permission)
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			if _, err := io.Copy(fileToWrite, tr); err != nil {
				return err
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	//
	return nil
}
