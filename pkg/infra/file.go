package infra

import (
	"os"
	"path/filepath"

	"github.com/m-mizutani/zenv/pkg/domain/types"
)

func (x *client) ReadFile(filename types.FilePath) ([]byte, error) {
	return os.ReadFile(filepath.Clean(string(filename)))
}
