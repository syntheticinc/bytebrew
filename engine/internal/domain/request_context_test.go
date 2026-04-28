package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRequestContext_StoreAndRetrieve(t *testing.T) {
	rc := &RequestContext{
		Headers: map[string]string{
			"X-Org-Id":  "org-123",
			"X-User-Id": "user-456",
		},
	}

	ctx := WithRequestContext(context.Background(), rc)
	got := GetRequestContext(ctx)

	require.NotNil(t, got)
	assert.Equal(t, "org-123", got.Headers["X-Org-Id"])
	assert.Equal(t, "user-456", got.Headers["X-User-Id"])
}

func TestGetRequestContext_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()
	got := GetRequestContext(ctx)
	assert.Nil(t, got)
}

func TestRequestContext_Get_CaseInsensitive(t *testing.T) {
	rc := &RequestContext{
		Headers: map[string]string{
			"X-Org-Id": "org-123",
		},
	}

	tests := []struct {
		name   string
		lookup string
		want   string
	}{
		{"exact case", "X-Org-Id", "org-123"},
		{"lower case", "x-org-id", "org-123"},
		{"upper case", "X-ORG-ID", "org-123"},
		{"mixed case", "x-Org-id", "org-123"},
		{"missing header", "X-Missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rc.Get(tt.lookup))
		})
	}
}

func TestRequestContext_Get_NilReceiver(t *testing.T) {
	var rc *RequestContext
	assert.Equal(t, "", rc.Get("X-Org-Id"))
}

func TestRequestContext_Get_NilHeaders(t *testing.T) {
	rc := &RequestContext{}
	assert.Equal(t, "", rc.Get("X-Org-Id"))
}

func TestWithRequestContext_OverwritesPrevious(t *testing.T) {
	rc1 := &RequestContext{Headers: map[string]string{"X-Org-Id": "org-1"}}
	rc2 := &RequestContext{Headers: map[string]string{"X-Org-Id": "org-2"}}

	ctx := WithRequestContext(context.Background(), rc1)
	ctx = WithRequestContext(ctx, rc2)

	got := GetRequestContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, "org-2", got.Get("X-Org-Id"))
}
