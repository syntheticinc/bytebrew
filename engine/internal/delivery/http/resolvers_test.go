package http

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- stub implementations ---

type stubAgentRefRepo struct {
	byID   map[string]string // id → id (verify exists)
	byName map[string]string // name → id
}

func (r *stubAgentRefRepo) GetAgentByID(_ context.Context, id string) (string, error) {
	if v, ok := r.byID[id]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

func (r *stubAgentRefRepo) GetAgentIDByName(_ context.Context, name string) (string, error) {
	if v, ok := r.byName[name]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

type stubModelRefRepo struct {
	byID   map[string]string
	byName map[string]string
}

func (r *stubModelRefRepo) GetModelByID(_ context.Context, id string) (string, error) {
	if v, ok := r.byID[id]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

func (r *stubModelRefRepo) GetModelIDByName(_ context.Context, name string) (string, error) {
	if v, ok := r.byName[name]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

// --- ResolveAgentRef tests ---

func TestResolveAgentRef_ByUUID_Found(t *testing.T) {
	repo := &stubAgentRefRepo{
		byID: map[string]string{"550e8400-e29b-41d4-a716-446655440000": "550e8400-e29b-41d4-a716-446655440000"},
	}
	id, err := ResolveAgentRef(context.Background(), repo, "550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id)
}

func TestResolveAgentRef_ByUUID_NotFound(t *testing.T) {
	repo := &stubAgentRefRepo{byID: map[string]string{}}
	_, err := ResolveAgentRef(context.Background(), repo, "550e8400-e29b-41d4-a716-446655440000")
	assert.ErrorIs(t, err, ErrRefNotFound)
}

func TestResolveAgentRef_ByName_Found(t *testing.T) {
	repo := &stubAgentRefRepo{
		byName: map[string]string{"my-agent": "550e8400-e29b-41d4-a716-446655440001"},
	}
	id, err := ResolveAgentRef(context.Background(), repo, "my-agent")
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", id)
}

func TestResolveAgentRef_ByName_NotFound(t *testing.T) {
	repo := &stubAgentRefRepo{byName: map[string]string{}}
	_, err := ResolveAgentRef(context.Background(), repo, "nonexistent")
	assert.ErrorIs(t, err, ErrRefNotFound)
}

func TestResolveAgentRef_NeverReturnsRefVerbatim(t *testing.T) {
	// Even if the UUID looks valid, we must DB round-trip — not return input verbatim.
	// Empty byID map means DB returns not-found → ErrRefNotFound.
	repo := &stubAgentRefRepo{byID: map[string]string{}}
	id, err := ResolveAgentRef(context.Background(), repo, "550e8400-e29b-41d4-a716-446655440000")
	assert.ErrorIs(t, err, ErrRefNotFound)
	assert.Empty(t, id)
}

// --- ResolveModelRef tests ---

func TestResolveModelRef_ByUUID_Found(t *testing.T) {
	repo := &stubModelRefRepo{
		byID: map[string]string{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
	}
	id, err := ResolveModelRef(context.Background(), repo, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	require.NoError(t, err)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", id)
}

func TestResolveModelRef_ByName_Found(t *testing.T) {
	repo := &stubModelRefRepo{
		byName: map[string]string{"gpt-4": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
	}
	id, err := ResolveModelRef(context.Background(), repo, "gpt-4")
	require.NoError(t, err)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", id)
}

func TestResolveModelRef_ByName_NotFound(t *testing.T) {
	repo := &stubModelRefRepo{byName: map[string]string{}}
	_, err := ResolveModelRef(context.Background(), repo, "missing-model")
	assert.ErrorIs(t, err, ErrRefNotFound)
}

// --- stubModelKindRepo (extends stubModelRefRepo with kind lookup) ---

type stubModelKindRepo struct {
	stubModelRefRepo
	kindByID map[string]string // id → kind
}

func (r *stubModelKindRepo) GetModelKindByID(_ context.Context, id string) (string, error) {
	if k, ok := r.kindByID[id]; ok {
		return k, nil
	}
	return "", errors.New("not found")
}

// --- ResolveModelRefWithKind tests ---

func TestResolveModelRefWithKind_ChatAccepted(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	repo := &stubModelKindRepo{
		stubModelRefRepo: stubModelRefRepo{
			byID: map[string]string{id: id},
		},
		kindByID: map[string]string{id: "chat"},
	}
	got, err := ResolveModelRefWithKind(context.Background(), repo, id, "chat")
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestResolveModelRefWithKind_RejectsEmbeddingWhereChatExpected(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	repo := &stubModelKindRepo{
		stubModelRefRepo: stubModelRefRepo{
			byID: map[string]string{id: id},
		},
		kindByID: map[string]string{id: "embedding"},
	}
	_, err := ResolveModelRefWithKind(context.Background(), repo, id, "chat")
	assert.ErrorIs(t, err, ErrModelKindMismatch)
}

func TestResolveModelRefWithKind_RejectsChatWhereEmbeddingExpected(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	repo := &stubModelKindRepo{
		stubModelRefRepo: stubModelRefRepo{
			byID: map[string]string{id: id},
		},
		kindByID: map[string]string{id: "chat"},
	}
	_, err := ResolveModelRefWithKind(context.Background(), repo, id, "embedding")
	assert.ErrorIs(t, err, ErrModelKindMismatch)
}

func TestResolveModelRefWithKind_EmbeddingAccepted(t *testing.T) {
	id := "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"
	repo := &stubModelKindRepo{
		stubModelRefRepo: stubModelRefRepo{
			byName: map[string]string{"text-embedding-3": id},
		},
		kindByID: map[string]string{id: "embedding"},
	}
	got, err := ResolveModelRefWithKind(context.Background(), repo, "text-embedding-3", "embedding")
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestResolveModelRefWithKind_RefNotFound(t *testing.T) {
	repo := &stubModelKindRepo{
		stubModelRefRepo: stubModelRefRepo{byID: map[string]string{}},
		kindByID:         map[string]string{},
	}
	_, err := ResolveModelRefWithKind(context.Background(), repo, "550e8400-e29b-41d4-a716-446655440000", "chat")
	assert.ErrorIs(t, err, ErrRefNotFound)
}
