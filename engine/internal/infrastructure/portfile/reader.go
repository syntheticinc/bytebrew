package portfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Reader читает port file из dataDir.
type Reader struct {
	path string
}

// NewReader создаёт Reader для чтения port file.
func NewReader(dataDir string) *Reader {
	return &Reader{
		path: filepath.Join(dataDir, fileName),
	}
}

// Read читает PortInfo из файла.
// Возвращает nil, nil если файл не существует.
// Возвращает nil, err если файл повреждён.
func (r *Reader) Read() (*PortInfo, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read port file: %w", err)
	}

	var info PortInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse port file: %w", err)
	}

	return &info, nil
}
