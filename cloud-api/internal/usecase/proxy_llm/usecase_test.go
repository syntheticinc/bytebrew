package proxy_llm

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSubReader struct {
	sub *domain.Subscription
	err error
}

func (m *mockSubReader) GetByUserID(_ context.Context, _ string) (*domain.Subscription, error) {
	return m.sub, m.err
}

type mockRateLimiter struct {
	err error
}

func (m *mockRateLimiter) Check(_ string) error {
	return m.err
}

type mockModelRouter struct {
	defaultModel  string
	roleOverrides map[string]string
}

func (m *mockModelRouter) RouteModel(role string) string {
	if model, ok := m.roleOverrides[role]; ok {
		return model
	}
	return m.defaultModel
}

func defaultMockModelRouter() *mockModelRouter {
	return &mockModelRouter{
		defaultModel: "zai-org/GLM-5",
		roleOverrides: map[string]string{
			"reviewer": "zai-org/GLM-4.7",
			"tester":   "zai-org/GLM-4.7",
		},
	}
}

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		role          string
		modelOverride string
		sub           *domain.Subscription
		subErr        error
		rateLimitErr  error
		wantModel     string
		wantErrCode   string
		wantErr       bool
	}{
		{
			name:    "empty user ID",
			userID:  "",
			wantErr: true,
			wantErrCode: errors.CodeInvalidInput,
		},
		{
			name:    "subscription lookup error",
			userID:  "user-1",
			subErr:  fmt.Errorf("db error"),
			wantErr: true,
			wantErrCode: errors.CodeInternal,
		},
		{
			name:    "no subscription",
			userID:  "user-1",
			sub:     nil,
			wantErr: true,
			wantErrCode: errors.CodeForbidden,
		},
		{
			name:   "inactive subscription",
			userID: "user-1",
			sub: &domain.Subscription{
				Tier:   domain.TierPersonal,
				Status: domain.StatusCanceled,
			},
			wantErr:     true,
			wantErrCode: errors.CodeForbidden,
		},
		{
			name:   "trial within rate limit",
			userID: "user-1",
			role:   "supervisor",
			sub: &domain.Subscription{
				Tier:   domain.TierTrial,
				Status: domain.StatusTrialing,
			},
			wantModel: "zai-org/GLM-5",
		},
		{
			name:   "trial rate limit exceeded",
			userID: "user-1",
			role:   "supervisor",
			sub: &domain.Subscription{
				Tier:   domain.TierTrial,
				Status: domain.StatusTrialing,
			},
			rateLimitErr: fmt.Errorf("rate limit exceeded: 20 steps/hour for trial"),
			wantErr:      true,
			wantErrCode:  "RATE_LIMITED",
		},
		{
			name:   "paid within quota",
			userID: "user-1",
			role:   "coder",
			sub: &domain.Subscription{
				Tier:            domain.TierPersonal,
				Status:          domain.StatusActive,
				ProxyStepsUsed:  100,
				ProxyStepsLimit: 300,
			},
			wantModel: "zai-org/GLM-5",
		},
		{
			name:   "paid quota exhausted",
			userID: "user-1",
			role:   "coder",
			sub: &domain.Subscription{
				Tier:            domain.TierPersonal,
				Status:          domain.StatusActive,
				ProxyStepsUsed:  300,
				ProxyStepsLimit: 300,
			},
			wantErr:     true,
			wantErrCode: "QUOTA_EXHAUSTED",
		},
		{
			name:          "model override used when provided",
			userID:        "user-1",
			role:          "supervisor",
			modelOverride: "custom/model-v2",
			sub: &domain.Subscription{
				Tier:            domain.TierPersonal,
				Status:          domain.StatusActive,
				ProxyStepsLimit: 300,
			},
			wantModel: "custom/model-v2",
		},
		{
			name:   "default routing reviewer",
			userID: "user-1",
			role:   "reviewer",
			sub: &domain.Subscription{
				Tier:            domain.TierTeams,
				Status:          domain.StatusActive,
				ProxyStepsLimit: 300,
			},
			wantModel: "zai-org/GLM-4.7",
		},
		{
			name:   "paid with zero limit allows unlimited",
			userID: "user-1",
			role:   "coder",
			sub: &domain.Subscription{
				Tier:            domain.TierPersonal,
				Status:          domain.StatusActive,
				ProxyStepsUsed:  500,
				ProxyStepsLimit: 0,
			},
			wantModel: "zai-org/GLM-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(
				&mockSubReader{sub: tt.sub, err: tt.subErr},
				&mockRateLimiter{err: tt.rateLimitErr},
				defaultMockModelRouter(),
			)

			result, err := uc.Authorize(context.Background(), tt.userID, tt.role, tt.modelOverride)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %q, got %q", tt.wantErrCode, errors.GetCode(err))
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result.Allowed)
			assert.Equal(t, tt.wantModel, result.TargetModel)
		})
	}
}
