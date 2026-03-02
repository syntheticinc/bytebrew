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
		Port:      60401,
		Host:      "localhost",
		StartedAt: "2026-03-01T10:00:00Z",
	}

	err := writer.Write(info)
	require.NoError(t, err)

	got, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, info.PID, got.PID)
	assert.Equal(t, info.Port, got.Port)
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
		Port:      60401,
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
		Port:      60401,
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
