package bridge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func validToken(token, code string) *domain.PairingToken {
	return &domain.PairingToken{
		Token:     token,
		ShortCode: code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
}

func expiredToken(token, code string) *domain.PairingToken {
	return &domain.PairingToken{
		Token:     token,
		ShortCode: code,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
}

func TestPairingTokenStore_AddAndGet(t *testing.T) {
	store := NewPairingTokenStore()
	tok := validToken("full-token", "123456")
	store.Add(tok)

	// Get by full token
	got := store.Get("full-token")
	require.NotNil(t, got)
	assert.Equal(t, "full-token", got.Token)

	// Get by short code
	got = store.Get("123456")
	require.NotNil(t, got)
	assert.Equal(t, "full-token", got.Token)
}

func TestPairingTokenStore_GetNotFound(t *testing.T) {
	store := NewPairingTokenStore()
	assert.Nil(t, store.Get("nonexistent"))
}

func TestPairingTokenStore_UseToken(t *testing.T) {
	store := NewPairingTokenStore()
	store.Add(validToken("tok-1", "111111"))

	// Use by short code
	got := store.UseToken("111111")
	require.NotNil(t, got)
	assert.True(t, got.Used)

	// Second use returns nil
	assert.Nil(t, store.UseToken("111111"))
}

func TestPairingTokenStore_UseToken_Expired(t *testing.T) {
	store := NewPairingTokenStore()
	store.Add(expiredToken("tok-1", "111111"))

	assert.Nil(t, store.UseToken("tok-1"))
}

func TestPairingTokenStore_UseToken_NotFound(t *testing.T) {
	store := NewPairingTokenStore()
	assert.Nil(t, store.UseToken("nonexistent"))
}

func TestPairingTokenStore_Remove(t *testing.T) {
	store := NewPairingTokenStore()
	store.Add(validToken("tok-1", "111111"))

	store.Remove("tok-1")

	assert.Nil(t, store.Get("tok-1"))
	assert.Nil(t, store.Get("111111"))
}

func TestPairingTokenStore_RemoveExpired(t *testing.T) {
	store := NewPairingTokenStore()
	store.Add(validToken("valid-1", "111111"))
	store.Add(expiredToken("expired-1", "222222"))
	store.Add(expiredToken("expired-2", "333333"))

	removed := store.RemoveExpired()
	assert.Equal(t, 2, removed)

	assert.NotNil(t, store.Get("valid-1"))
	assert.Nil(t, store.Get("expired-1"))
	assert.Nil(t, store.Get("expired-2"))
}

func TestPairingTokenStore_AddWithoutShortCode(t *testing.T) {
	store := NewPairingTokenStore()
	store.Add(validToken("tok-1", ""))

	got := store.Get("tok-1")
	require.NotNil(t, got)
}
