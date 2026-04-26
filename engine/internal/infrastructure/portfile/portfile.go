package portfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const fileName = "server.port"

// PortInfo содержит информацию о запущенном сервере.
type PortInfo struct {
	PID          int    `json:"pid"`
	HTTPPort     int    `json:"http_port,omitempty"`     // External HTTP (data plane)
	InternalPort int    `json:"internal_port,omitempty"` // Internal HTTP (control plane), 0 = single-port
	Host         string `json:"host"`
	StartedAt    string `json:"startedAt"`
}

// Writer записывает port file в dataDir.
type Writer struct {
	path string
}

// NewWriter создаёт Writer. Port file будет по пути: dataDir/server.port.
func NewWriter(dataDir string) *Writer {
	return &Writer{
		path: filepath.Join(dataDir, fileName),
	}
}

// Write записывает PortInfo в файл атомарно (write tmp → rename).
func (w *Writer) Write(info PortInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal port info: %w", err)
	}

	tmpPath := w.path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write tmp port file: %w", err)
	}

	if err := os.Rename(tmpPath, w.path); err != nil {
		// Cleanup tmp file on rename failure.
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename port file: %w", err)
	}

	return nil
}

// Remove удаляет port file (вызывается при graceful shutdown).
func (w *Writer) Remove() error {
	if err := os.Remove(w.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove port file: %w", err)
	}
	return nil
}

// Path возвращает путь к port file.
func (w *Writer) Path() string {
	return w.path
}
