package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPairingToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"future", time.Now().Add(10 * time.Minute), false},
		{"past", time.Now().Add(-1 * time.Minute), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := PairingToken{Token: "t", ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.want, token.IsExpired())
		})
	}
}

func TestPairingToken_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		used      bool
		want      bool
	}{
		{"valid", time.Now().Add(10 * time.Minute), false, true},
		{"expired", time.Now().Add(-1 * time.Minute), false, false},
		{"used", time.Now().Add(10 * time.Minute), true, false},
		{"used and expired", time.Now().Add(-1 * time.Minute), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := PairingToken{Token: "t", ExpiresAt: tt.expiresAt, Used: tt.used}
			assert.Equal(t, tt.want, token.IsValid())
		})
	}
}

func TestPairingToken_MarkUsed(t *testing.T) {
	token := PairingToken{Token: "t", ExpiresAt: time.Now().Add(10 * time.Minute)}
	assert.True(t, token.IsValid())

	token.MarkUsed()
	assert.True(t, token.Used)
	assert.False(t, token.IsValid())
}
