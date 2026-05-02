package authprim

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_FormatStable(t *testing.T) {
	tok, err := Generate()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(tok, Prefix))
	assert.Len(t, tok, Length)
	require.NoError(t, ValidateFormat(tok))
}

func TestGenerate_NotConstant(t *testing.T) {
	a, err := Generate()
	require.NoError(t, err)
	b, err := Generate()
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "two generated tokens must differ")
}

// TestGenerate_NotAllZero defends against a class of broken-RNG environments
// (sealed CI containers, /dev/urandom not exposed) where rand.Read may
// silently return zeros. crypto/rand returns an error in that case, but a
// belt-and-suspenders smoke prevents a downstream "valid format, no entropy"
// admin token sneaking past ValidateFormat.
func TestGenerate_NotAllZero(t *testing.T) {
	tok, err := Generate()
	require.NoError(t, err)
	allZero := Prefix + strings.Repeat("0", HexLen)
	assert.NotEqual(t, allZero, tok, "RNG produced all-zero token — environment likely lacks entropy")
}

func TestHash_StableAndDistinct(t *testing.T) {
	h1 := Hash("bb_test")
	h2 := Hash("bb_test")
	h3 := Hash("bb_other")
	assert.Equal(t, h1, h2, "same input must hash identically")
	assert.NotEqual(t, h1, h3, "different inputs must hash distinctly")
	assert.Len(t, h1, 64, "SHA-256 hex is 64 chars")
}

func TestValidateFormat(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"no prefix", "abcd1234", true},
		{"wrong prefix", "aa_" + strings.Repeat("0", 64), true},
		{"too short", "bb_" + strings.Repeat("0", 63), true},
		{"too long", "bb_" + strings.Repeat("0", 65), true},
		{"non-hex chars", "bb_" + strings.Repeat("z", 64), true},
		{"valid all zeros", "bb_" + strings.Repeat("0", 64), false},
		{"valid generated", mustGenerate(t), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFormat(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidTokenFormat),
					"every invalid-format error must wrap ErrInvalidTokenFormat for errors.Is matching")
				return
			}
			assert.NoError(t, err)
		})
	}
}

func mustGenerate(t *testing.T) string {
	t.Helper()
	tok, err := Generate()
	require.NoError(t, err)
	return tok
}
