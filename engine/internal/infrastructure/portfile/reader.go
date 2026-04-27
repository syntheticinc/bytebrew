package portfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Reader reads the port file from dataDir.
type Reader struct {
	path string
}

// NewReader creates a Reader for reading the port file.
func NewReader(dataDir string) *Reader {
	return &Reader{
		path: filepath.Join(dataDir, fileName),
	}
}

// Read reads PortInfo from disk.
// Returns nil, nil if the file does not exist.
// Returns nil, err if the file is corrupted.
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
