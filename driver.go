package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Driver struct {
	rootPath string
}

type FileInfo struct {
}

func (d *Driver) realPath(path string) string {
	paths := strings.Split(path, "/")
	return filepath.Join(append([]string{d.rootPath}, paths...)...)
}

func (d *Driver) PutFile(destPath string, data io.Reader, appendData bool) (int64, error) {
	rPath := d.realPath(destPath)
	var isExist bool
	f, err := os.Lstat(rPath)
	if err == nil {
		isExist = true
		if f.IsDir() {
			return 0, errors.New("a dir has the same name")
		}
	} else {
		if os.IsNotExist(err) {
			isExist = false
		} else {
			return 0, errors.New(fmt.Sprintf("put file error: %s", err))
		}
	}

	if appendData && !isExist {
		appendData = false
	}
	if !appendData {
		if isExist {
			err = os.Remove(rPath)
			if err != nil {
				return 0, err
			}
		}
		f, err := os.Create(rPath)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		bytes, err := io.Copy(f, data)
		if err != nil {
			return 0, err
		}
		return bytes, nil
	}

	of, err := os.OpenFile(rPath, os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		return 0, err
	}
	defer of.Close()
	_, err = of.Seek(0, os.SEEK_END)
	if err != nil {
		return 0, err
	}
	bytes, err := io.Copy(of, data)
	if err != nil {
		return 0, err
	}
	return bytes, nil
}
