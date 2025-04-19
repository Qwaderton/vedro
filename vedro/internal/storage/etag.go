package storage

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func ComputeETag(path string, isDir bool) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	hash := md5.New()

	hash.Write([]byte(path))
	hash.Write([]byte(info.ModTime().UTC().String()))

	if !isDir {
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash.Sum(nil))), nil
}