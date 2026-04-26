package portfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)
	reader := NewReader(dir)

	info := PortInfo{
		PID:       12345,
		HTTPPort:  8443,
		Host:      "localhost",
		StartedAt: "2026-03-01T10:00:00Z",
	}

	err := writer.Write(info)
	require.NoError(t, err)

	got, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, info.PID, got.PID)
	assert.Equal(t, info.HTTPPort, got.HTTPPort)
	assert.Equal(t, info.Host, got.Host)
	assert.Equal(t, info.StartedAt, got.StartedAt)
}

func TestReadNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	reader := NewReader(dir)

	got, err := reader.Read()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestReadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)

	err := os.WriteFile(path, []byte("not valid json{{{"), 0644)
	require.NoError(t, err)

	reader := NewReader(dir)
	got, err := reader.Read()
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse port file")
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)
	reader := NewReader(dir)

	info := PortInfo{
		PID:       12345,
		HTTPPort:  8443,
		Host:      "localhost",
		StartedAt: "2026-03-01T10:00:00Z",
	}

	err := writer.Write(info)
	require.NoError(t, err)

	err = writer.Remove()
	require.NoError(t, err)

	got, err := reader.Read()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRemoveNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)

	err := writer.Remove()
	require.NoError(t, err)
}

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	assert.True(t, IsProcessAlive(pid))
}

func TestIsProcessAlive_DeadProcess(t *testing.T) {
	// PID 99999999 практически наверняка не существует.
	assert.False(t, IsProcessAlive(99999999))
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	assert.False(t, IsProcessAlive(0))
	assert.False(t, IsProcessAlive(-1))
}

func TestAtomicWrite_NoTmpFileRemains(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)

	info := PortInfo{
		PID:       12345,
		HTTPPort:  8443,
		Host:      "localhost",
		StartedAt: "2026-03-01T10:00:00Z",
	}

	err := writer.Write(info)
	require.NoError(t, err)

	tmpPath := filepath.Join(dir, fileName+".tmp")
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "tmp file should not remain after successful write")
}

func TestWriterPath(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)

	expected := filepath.Join(dir, fileName)
	assert.Equal(t, expected, writer.Path())
}

func TestPortInfo_WithHTTPAndInternalPort(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)
	reader := NewReader(dir)

	info := PortInfo{
		PID:          12345,
		HTTPPort:     8443,
		InternalPort: 8444,
		Host:         "0.0.0.0",
		StartedAt:    "2026-03-29T10:00:00Z",
	}

	err := writer.Write(info)
	require.NoError(t, err)

	got, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 12345, got.PID)
	assert.Equal(t, 8443, got.HTTPPort)
	assert.Equal(t, 8444, got.InternalPort)
	assert.Equal(t, "0.0.0.0", got.Host)
}

func TestPortInfo_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)

	// Write old-format JSON without http_port/internal_port fields.
	// (legacy "port" field is ignored on read since it has no struct member.)
	oldJSON := `{"pid":12345,"port":50051,"host":"localhost","startedAt":"2026-03-29T10:00:00Z"}`
	require.NoError(t, os.WriteFile(path, []byte(oldJSON), 0644))

	reader := NewReader(dir)
	got, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 12345, got.PID)
	assert.Equal(t, "localhost", got.Host)
	// New fields default to zero.
	assert.Equal(t, 0, got.HTTPPort)
	assert.Equal(t, 0, got.InternalPort)
}

func TestPortInfo_OmitEmpty(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)

	info := PortInfo{
		PID:       12345,
		HTTPPort:  8443,
		Host:      "localhost",
		StartedAt: "2026-03-29T10:00:00Z",
		// InternalPort is 0 — should be omitted from JSON.
	}

	err := writer.Write(info)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, fileName))
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "internal_port")
	assert.NotContains(t, jsonStr, "ws_port")
	assert.NotContains(t, jsonStr, `"port"`)

	// Fields that are always present.
	assert.Contains(t, jsonStr, `"pid"`)
	assert.Contains(t, jsonStr, `"http_port"`)
	assert.Contains(t, jsonStr, `"host"`)
}

func TestPortInfo_WithAllPorts(t *testing.T) {
	dir := t.TempDir()
	writer := NewWriter(dir)
	reader := NewReader(dir)

	info := PortInfo{
		PID:          99999,
		HTTPPort:     8443,
		InternalPort: 8444,
		Host:         "127.0.0.1",
		StartedAt:    "2026-03-29T12:00:00Z",
	}

	require.NoError(t, writer.Write(info))

	got, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, info.PID, got.PID)
	assert.Equal(t, info.HTTPPort, got.HTTPPort)
	assert.Equal(t, info.InternalPort, got.InternalPort)
	assert.Equal(t, info.Host, got.Host)
	assert.Equal(t, info.StartedAt, got.StartedAt)
}
