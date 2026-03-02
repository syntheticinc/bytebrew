package mobile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func TestPairingWaiter_NotifyBeforeRegister(t *testing.T) {
	w := NewPairingWaiter()

	// Notify without anyone waiting should not panic
	w.Notify("token-1", domain.PairingNotification{
		DeviceName: "iPhone",
		DeviceID:   "dev-1",
	})
}

func TestPairingWaiter_RegisterAndNotify(t *testing.T) {
	w := NewPairingWaiter()

	ch := w.Register("token-1")

	go func() {
		time.Sleep(10 * time.Millisecond)
		w.Notify("token-1", domain.PairingNotification{
			DeviceName: "iPhone 15",
			DeviceID:   "dev-1",
		})
	}()

	select {
	case result := <-ch:
		assert.Equal(t, "iPhone 15", result.DeviceName)
		assert.Equal(t, "dev-1", result.DeviceID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pairing notification")
	}
}

func TestPairingWaiter_UnregisterCleansUp(t *testing.T) {
	w := NewPairingWaiter()

	ch := w.Register("token-1")
	w.Unregister("token-1")

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after Unregister")
}

func TestPairingWaiter_UnregisterIdempotent(t *testing.T) {
	w := NewPairingWaiter()

	w.Register("token-1")
	w.Unregister("token-1")
	// Second unregister should not panic
	w.Unregister("token-1")
}

func TestPairingWaiter_NotifyRemovesWaiter(t *testing.T) {
	w := NewPairingWaiter()

	ch := w.Register("token-1")

	w.Notify("token-1", domain.PairingNotification{
		DeviceName: "iPhone",
		DeviceID:   "dev-1",
	})

	// Read the notification
	result := <-ch
	assert.Equal(t, "iPhone", result.DeviceName)

	// Second notify should be silently dropped (waiter already removed)
	w.Notify("token-1", domain.PairingNotification{
		DeviceName: "Pixel",
		DeviceID:   "dev-2",
	})
}

func TestPairingWaiter_MultipleTokens(t *testing.T) {
	w := NewPairingWaiter()

	ch1 := w.Register("token-1")
	ch2 := w.Register("token-2")

	w.Notify("token-2", domain.PairingNotification{
		DeviceName: "Pixel 8",
		DeviceID:   "dev-2",
	})

	select {
	case result := <-ch2:
		assert.Equal(t, "Pixel 8", result.DeviceName)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token-2 notification")
	}

	// token-1 should still be waiting (no notification sent)
	select {
	case <-ch1:
		t.Fatal("token-1 should not have received a notification")
	case <-time.After(50 * time.Millisecond):
		// Expected: no notification for token-1
	}

	w.Unregister("token-1")
}

func TestPairingWaiter_ImplementsPairingWaiterService(t *testing.T) {
	// Compile-time check that PairingWaiter implements the consumer-side interface
	// defined in delivery/grpc/mobile_handler.go
	w := NewPairingWaiter()

	// Verify the methods exist with correct signatures
	var ch <-chan domain.PairingNotification
	ch = w.Register("test-token")
	require.NotNil(t, ch)

	w.Notify("test-token", domain.PairingNotification{DeviceName: "test", DeviceID: "id"})
	w.Unregister("other-token")
}
