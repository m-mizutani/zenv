package infra

import (
	"io/ioutil"
	"path/filepath"
)

func (x *Infrastructure) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Clean(filename))
}
