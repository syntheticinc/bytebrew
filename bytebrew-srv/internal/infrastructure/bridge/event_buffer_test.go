package bridge

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBuffer_Append(t *testing.T) {
	buf := NewEventBuffer(10)

	id := buf.Append("session-1", map[string]interface{}{"type": "test"})
	assert.Equal(t, "mevt-1", id)

	id2 := buf.Append("session-1", map[string]interface{}{"type": "test2"})
	assert.Equal(t, "mevt-2", id2)
}

func TestEventBuffer_GetAfter_EmptyID(t *testing.T) {
	buf := NewEventBuffer(10)
	buf.Append("s1", map[string]interface{}{"a": 1})

	result := buf.GetAfter("")
	assert.Nil(t, result)
}

func TestEventBuffer_GetAfter_ReturnsSubsequentEvents(t *testing.T) {
	buf := NewEventBuffer(10)
	buf.Append("s1", map[string]interface{}{"n": 1})
	buf.Append("s1", map[string]interface{}{"n": 2})
	buf.Append("s1", map[string]interface{}{"n": 3})

	result := buf.GetAfter("mevt-1")
	require.Len(t, result, 2)
	assert.Equal(t, "mevt-2", result[0].EventID)
	assert.Equal(t, "mevt-3", result[1].EventID)
}

func TestEventBuffer_GetAfter_LastEvent(t *testing.T) {
	buf := NewEventBuffer(10)
	buf.Append("s1", map[string]interface{}{"n": 1})
	buf.Append("s1", map[string]interface{}{"n": 2})

	result := buf.GetAfter("mevt-2")
	assert.Empty(t, result)
}

func TestEventBuffer_GetAfter_NotFound(t *testing.T) {
	buf := NewEventBuffer(10)
	buf.Append("s1", map[string]interface{}{"n": 1})

	result := buf.GetAfter("mevt-999")
	assert.Nil(t, result)
}

func TestEventBuffer_RingBuffer_OverwritesOldest(t *testing.T) {
	buf := NewEventBuffer(3)

	buf.Append("s1", map[string]interface{}{"n": 1}) // mevt-1
	buf.Append("s1", map[string]interface{}{"n": 2}) // mevt-2
	buf.Append("s1", map[string]interface{}{"n": 3}) // mevt-3
	buf.Append("s1", map[string]interface{}{"n": 4}) // mevt-4, overwrites mevt-1

	// mevt-1 is gone, so GetAfter("mevt-1") should find nothing.
	result := buf.GetAfter("mevt-1")
	assert.Nil(t, result)

	// mevt-2 is still there.
	result = buf.GetAfter("mevt-2")
	require.Len(t, result, 2)
	assert.Equal(t, "mevt-3", result[0].EventID)
	assert.Equal(t, "mevt-4", result[1].EventID)
}

func TestEventBuffer_DefaultSize(t *testing.T) {
	buf := NewEventBuffer(0)
	assert.Len(t, buf.events, 1000)
}

func TestEventBuffer_ConcurrentAccess(t *testing.T) {
	buf := NewEventBuffer(100)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				buf.Append("s1", map[string]interface{}{"n": n*10 + j})
			}
		}(i)
	}

	// Concurrent reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			buf.GetAfter(fmt.Sprintf("mevt-%d", i+1))
		}
	}()

	wg.Wait()
}
