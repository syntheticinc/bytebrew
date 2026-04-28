package callbacks

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepCounter_InitialValues(t *testing.T) {
	c := NewStepCounter()
	assert.Equal(t, 0, c.GetStep())
	assert.Equal(t, 0, c.GetModelCallCount())
}

func TestStepCounter_IncrementStep(t *testing.T) {
	c := NewStepCounter()
	ctx := context.Background()

	assert.NoError(t, c.IncrementStep(ctx))
	assert.Equal(t, 1, c.GetStep())

	assert.NoError(t, c.IncrementStep(ctx))
	assert.NoError(t, c.IncrementStep(ctx))
	assert.Equal(t, 3, c.GetStep())
}

func TestStepCounter_IncrementStep_ThreadSafe(t *testing.T) {
	c := NewStepCounter()
	ctx := context.Background()

	var wg sync.WaitGroup
	numIncrements := 100
	wg.Add(numIncrements)

	for i := 0; i < numIncrements; i++ {
		go func() {
			defer wg.Done()
			_ = c.IncrementStep(ctx)
		}()
	}

	wg.Wait()
	assert.Equal(t, numIncrements, c.GetStep())
}

func TestStepCounter_ModelCallCount(t *testing.T) {
	c := NewStepCounter()

	c.IncrementModelCallCount()
	assert.Equal(t, 1, c.GetModelCallCount())

	c.IncrementModelCallCount()
	assert.Equal(t, 2, c.GetModelCallCount())
}

func TestStepCounter_PendingAssistantContent(t *testing.T) {
	c := NewStepCounter()

	// Initially empty
	assert.Equal(t, "", c.ConsumePendingAssistantContent())

	// Set and consume
	c.SetPendingAssistantContent("Hello, world!")
	assert.Equal(t, "Hello, world!", c.ConsumePendingAssistantContent())

	// Consumed - should be empty now
	assert.Equal(t, "", c.ConsumePendingAssistantContent())
}

func TestStepCounter_PendingAssistantContent_Overwrite(t *testing.T) {
	c := NewStepCounter()

	c.SetPendingAssistantContent("first")
	c.SetPendingAssistantContent("second")
	assert.Equal(t, "second", c.ConsumePendingAssistantContent())
}

func TestStepCounter_StepCallback_Fires(t *testing.T) {
	// Reset at end so other tests are not affected.
	t.Cleanup(func() { SetStepCallback(nil) })

	var calls int32
	SetStepCallback(func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	c := NewStepCounter()
	assert.NoError(t, c.IncrementStep(context.Background()))
	assert.NoError(t, c.IncrementStep(context.Background()))

	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestStepCounter_StepCallback_Nil_NoCrash(t *testing.T) {
	// Ensure any callback installed by a previous test is cleared.
	SetStepCallback(nil)

	c := NewStepCounter()
	// Must not panic when callback is unset.
	assert.NoError(t, c.IncrementStep(context.Background()))
	assert.Equal(t, 1, c.GetStep())
}
