package callbacks

import (
	"sync"
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

	c.IncrementStep()
	assert.Equal(t, 1, c.GetStep())

	c.IncrementStep()
	c.IncrementStep()
	assert.Equal(t, 3, c.GetStep())
}

func TestStepCounter_IncrementStep_ThreadSafe(t *testing.T) {
	c := NewStepCounter()

	var wg sync.WaitGroup
	numIncrements := 100
	wg.Add(numIncrements)

	for i := 0; i < numIncrements; i++ {
		go func() {
			defer wg.Done()
			c.IncrementStep()
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
