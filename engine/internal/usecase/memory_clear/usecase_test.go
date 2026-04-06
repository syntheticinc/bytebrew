package memory_clear

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	deletedCount int64
	deleteErr    error
}

func (m *mockRepo) DeleteBySchema(ctx context.Context, schemaID string) (int64, error) {
	return m.deletedCount, m.deleteErr
}

func (m *mockRepo) DeleteByID(ctx context.Context, id string) error {
	return m.deleteErr
}

func TestUsecase_ClearAll(t *testing.T) {
	repo := &mockRepo{deletedCount: 5}
	uc := New(repo)

	deleted, err := uc.ClearAll(context.Background(), "10")
	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted)
}

func TestUsecase_ClearAll_EmptySchemaID(t *testing.T) {
	uc := New(&mockRepo{})
	_, err := uc.ClearAll(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema_id is required")
}

func TestUsecase_ClearAll_RepoError(t *testing.T) {
	repo := &mockRepo{deleteErr: fmt.Errorf("db error")}
	uc := New(repo)
	_, err := uc.ClearAll(context.Background(), "10")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clear memories")
}

func TestUsecase_DeleteOne(t *testing.T) {
	uc := New(&mockRepo{})
	err := uc.DeleteOne(context.Background(), "42")
	require.NoError(t, err)
}

func TestUsecase_DeleteOne_EmptyID(t *testing.T) {
	uc := New(&mockRepo{})
	err := uc.DeleteOne(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory id is required")
}

func TestUsecase_DeleteOne_RepoError(t *testing.T) {
	repo := &mockRepo{deleteErr: fmt.Errorf("not found")}
	uc := New(repo)
	err := uc.DeleteOne(context.Background(), "42")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete memory")
}
