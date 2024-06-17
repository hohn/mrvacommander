package utils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// UnzipFile extracts a zip file to the specified destination
func UnzipFile(zipFile, dest string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(dest, f.Name)

		// mitigate ZipSlip
		if !strings.HasPrefix(filepath.Clean(fPath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fPath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// UntarGz extracts a tar.gz file to the specified destination.
func UntarGz(tarGzFile, dest string) error {
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return Untar(gzr, dest)
}

// Untar extracts a tar archive to the specified destination.
func Untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fPath := filepath.Join(dest, header.Name)

		// mitigate ZipSlip
		if !strings.HasPrefix(filepath.Clean(fPath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fPath)
		}

		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}

			outFile.Close()
		}
	}

	return nil
}

// IsBase64Gzip checks the request body up to the `gunzip` part.
//
// Some important payloads can be listed via
// base64 -d < foo1 | gunzip | tar t|head -20
func IsBase64Gzip(val []byte) bool {
	if len(val) >= 4 {
		// Extract header
		hdr := make([]byte, base64.StdEncoding.DecodedLen(4))
		_, err := base64.StdEncoding.Decode(hdr, []byte(val[0:4]))
		if err != nil {
			log.Println("WARNING: IsBase64Gzip decode error:", err)
			return false
		}
		// Check for gzip heading
		magic := []byte{0x1f, 0x8b}
		if bytes.Equal(hdr[0:2], magic) {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}
