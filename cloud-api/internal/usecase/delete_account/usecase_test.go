package delete_account

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// --- Mocks ---

type mockUserReader struct {
	users map[string]*domain.User
	err   error
}

func newMockUserReader() *mockUserReader {
	return &mockUserReader{users: make(map[string]*domain.User)}
}

func (m *mockUserReader) GetByID(_ context.Context, id string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users[id], nil
}

func (m *mockUserReader) addUser(id, password string) {
	m.users[id] = &domain.User{
		ID:           id,
		Email:        id + "@example.com",
		PasswordHash: "hashed-" + password,
	}
}

type mockUserDeleter struct {
	deleted []string
	err     error
}

func (m *mockUserDeleter) Delete(_ context.Context, userID string) error {
	if m.err != nil {
		return m.err
	}
	m.deleted = append(m.deleted, userID)
	return nil
}

type mockPasswordHasher struct {
	compareErr error
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	if m.compareErr != nil {
		return m.compareErr
	}
	if hash == "hashed-"+password {
		return nil
	}
	return fmt.Errorf("password mismatch")
}

type mockSubReader struct {
	subs map[string]*domain.Subscription
	err  error
}

func (m *mockSubReader) GetByUserID(_ context.Context, userID string) (*domain.Subscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.subs[userID], nil
}

type mockSubCanceller struct {
	cancelled []string
	err       error
}

func (m *mockSubCanceller) CancelSubscription(_ context.Context, stripeSubID string) error {
	if m.err != nil {
		return m.err
	}
	m.cancelled = append(m.cancelled, stripeSubID)
	return nil
}

// --- Tests ---

func strPtr(s string) *string { return &s }

func TestExecute(t *testing.T) {
	tests := []struct {
		name           string
		input          Input
		setupRepo      func(*mockUserReader)
		subReaderSubs  map[string]*domain.Subscription
		subReaderErr   error
		cancelErr      error
		deleterErr     error
		wantCode       string
		wantDeleted    bool
		wantCancelled  string // expected stripe subscription ID cancelled
		wantNoCancell  bool   // expect no cancellation call
	}{
		{
			name:  "happy path with stripe subscription",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderSubs: map[string]*domain.Subscription{
				"user-1": {StripeSubscriptionID: strPtr("sub_stripe_123")},
			},
			wantDeleted:   true,
			wantCancelled: "sub_stripe_123",
		},
		{
			name:  "happy path without subscription",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderSubs: map[string]*domain.Subscription{},
			wantDeleted:   true,
			wantNoCancell: true,
		},
		{
			name:  "happy path with nil stripe subscription ID",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderSubs: map[string]*domain.Subscription{
				"user-1": {StripeSubscriptionID: nil},
			},
			wantDeleted:   true,
			wantNoCancell: true,
		},
		{
			name:  "happy path with empty stripe subscription ID",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderSubs: map[string]*domain.Subscription{
				"user-1": {StripeSubscriptionID: strPtr("")},
			},
			wantDeleted:   true,
			wantNoCancell: true,
		},
		{
			name:     "user not found",
			input:    Input{UserID: "nobody", Password: "secret"},
			wantCode: errors.CodeNotFound,
		},
		{
			name:  "wrong password",
			input: Input{UserID: "user-1", Password: "wrong"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			wantCode: errors.CodeUnauthorized,
		},
		{
			name:  "subscription reader error does not block deletion",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderErr:  fmt.Errorf("db connection lost"),
			wantDeleted:   true,
			wantNoCancell: true,
		},
		{
			name:  "cancel subscription error does not block deletion",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderSubs: map[string]*domain.Subscription{
				"user-1": {StripeSubscriptionID: strPtr("sub_stripe_456")},
			},
			cancelErr:   fmt.Errorf("stripe API error"),
			wantDeleted: true,
		},
		{
			name:  "deleter error returns internal error",
			input: Input{UserID: "user-1", Password: "secret"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "secret")
			},
			subReaderSubs: map[string]*domain.Subscription{},
			deleterErr:    fmt.Errorf("db error"),
			wantCode:      errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newMockUserReader()
			if tt.setupRepo != nil {
				tt.setupRepo(reader)
			}

			deleter := &mockUserDeleter{err: tt.deleterErr}
			hasher := &mockPasswordHasher{}
			subReader := &mockSubReader{subs: tt.subReaderSubs, err: tt.subReaderErr}
			canceller := &mockSubCanceller{err: tt.cancelErr}

			uc := New(reader, deleter, hasher, subReader, canceller)
			err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				if err == nil {
					t.Fatalf("expected error with code %s, got nil", tt.wantCode)
				}
				if !errors.Is(err, tt.wantCode) {
					t.Fatalf("expected error code %s, got: %v", tt.wantCode, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantDeleted && len(deleter.deleted) == 0 {
				t.Fatal("expected user to be deleted")
			}

			if tt.wantCancelled != "" {
				if len(canceller.cancelled) == 0 {
					t.Fatalf("expected subscription %s to be cancelled", tt.wantCancelled)
				}
				if canceller.cancelled[0] != tt.wantCancelled {
					t.Fatalf("cancelled subscription = %q, want %q", canceller.cancelled[0], tt.wantCancelled)
				}
			}

			if tt.wantNoCancell && len(canceller.cancelled) > 0 {
				t.Fatalf("expected no cancellation, got: %v", canceller.cancelled)
			}
		})
	}
}
