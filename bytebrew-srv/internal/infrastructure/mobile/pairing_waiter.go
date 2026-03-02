package mobile

import (
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// PairingWaiter allows callers to wait for a specific pairing token to be consumed.
// CLI calls Register(token) before showing the QR code, then selects on the returned
// channel. When a mobile device calls Pair() with that token, the handler calls
// Notify() which delivers the result to the waiting CLI.
type PairingWaiter struct {
	mu      sync.Mutex
	waiters map[string]chan domain.PairingNotification
}

// NewPairingWaiter creates a new PairingWaiter.
func NewPairingWaiter() *PairingWaiter {
	return &PairingWaiter{
		waiters: make(map[string]chan domain.PairingNotification),
	}
}

// Register creates a buffered channel that will receive the PairingNotification when
// the token is consumed. Caller should call Unregister when done to clean up.
func (w *PairingWaiter) Register(token string) <-chan domain.PairingNotification {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan domain.PairingNotification, 1)
	w.waiters[token] = ch
	return ch
}

// Notify sends the pairing notification to anyone waiting for this token.
// Safe to call even if no one is waiting (the notification is silently dropped).
func (w *PairingWaiter) Notify(token string, notification domain.PairingNotification) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch, ok := w.waiters[token]
	if !ok {
		return
	}

	select {
	case ch <- notification:
	default:
	}

	delete(w.waiters, token)
}

// Unregister removes the waiter for the given token and closes the channel.
// Safe to call multiple times or if the token was already notified.
func (w *PairingWaiter) Unregister(token string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch, ok := w.waiters[token]
	if !ok {
		return
	}

	close(ch)
	delete(w.waiters, token)
}
