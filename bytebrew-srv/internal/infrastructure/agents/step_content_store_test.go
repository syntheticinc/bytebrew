package agents

import (
	"sync"
	"testing"
)

func TestStepContentStore_Append(t *testing.T) {
	store := NewStepContentStore()

	// Test single append
	store.Append(0, "Hello")
	if got := store.Get(0); got != "Hello" {
		t.Errorf("Append single: got %q, want %q", got, "Hello")
	}

	// Test multiple appends to same step
	store.Append(0, " World")
	if got := store.Get(0); got != "Hello World" {
		t.Errorf("Append multiple: got %q, want %q", got, "Hello World")
	}

	// Test append to different steps
	store.Append(1, "Step 1 content")
	store.Append(2, "Step 2 content")

	if got := store.Get(1); got != "Step 1 content" {
		t.Errorf("Append step 1: got %q, want %q", got, "Step 1 content")
	}
	if got := store.Get(2); got != "Step 2 content" {
		t.Errorf("Append step 2: got %q, want %q", got, "Step 2 content")
	}
}

func TestStepContentStore_Get(t *testing.T) {
	store := NewStepContentStore()

	// Test get non-existent step returns empty string
	if got := store.Get(999); got != "" {
		t.Errorf("Get non-existent: got %q, want empty string", got)
	}

	// Test get existing step
	store.Append(5, "test content")
	if got := store.Get(5); got != "test content" {
		t.Errorf("Get existing: got %q, want %q", got, "test content")
	}
}

func TestStepContentStore_GetAll(t *testing.T) {
	store := NewStepContentStore()

	// Test empty store
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("GetAll empty: got %d items, want 0", len(all))
	}

	// Populate store
	store.Append(0, "step 0")
	store.Append(1, "step 1")
	store.Append(2, "step 2")

	all = store.GetAll()
	if len(all) != 3 {
		t.Errorf("GetAll populated: got %d items, want 3", len(all))
	}

	// Verify values
	if all[0] != "step 0" {
		t.Errorf("GetAll step 0: got %q, want %q", all[0], "step 0")
	}
	if all[1] != "step 1" {
		t.Errorf("GetAll step 1: got %q, want %q", all[1], "step 1")
	}
	if all[2] != "step 2" {
		t.Errorf("GetAll step 2: got %q, want %q", all[2], "step 2")
	}

	// Verify GetAll returns a copy (modifying returned map doesn't affect store)
	all[0] = "modified"
	if got := store.Get(0); got != "step 0" {
		t.Errorf("GetAll copy: original was modified, got %q, want %q", got, "step 0")
	}
}

func TestStepContentStore_ThreadSafety(t *testing.T) {
	store := NewStepContentStore()
	var wg sync.WaitGroup

	// Concurrent writes to same step
	numWriters := 100
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(idx int) {
			defer wg.Done()
			store.Append(0, "x")
		}(i)
	}
	wg.Wait()

	content := store.Get(0)
	if len(content) != numWriters {
		t.Errorf("ThreadSafety writes: got length %d, want %d", len(content), numWriters)
	}

	// Concurrent writes to different steps
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(idx int) {
			defer wg.Done()
			store.Append(idx+1, "content")
		}(i)
	}
	wg.Wait()

	all := store.GetAll()
	// Should have step 0 plus 100 additional steps
	if len(all) != numWriters+1 {
		t.Errorf("ThreadSafety different steps: got %d steps, want %d", len(all), numWriters+1)
	}

	// Concurrent reads and writes
	wg.Add(numWriters * 2)
	for i := 0; i < numWriters; i++ {
		go func(idx int) {
			defer wg.Done()
			store.Append(200, "y")
		}(i)
		go func(idx int) {
			defer wg.Done()
			_ = store.Get(200)
		}(i)
	}
	wg.Wait()

	// Should complete without race condition
	content200 := store.Get(200)
	if len(content200) != numWriters {
		t.Errorf("ThreadSafety read/write: got length %d, want %d", len(content200), numWriters)
	}

	// Concurrent GetAll calls
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			_ = store.GetAll()
		}()
	}
	wg.Wait()
	// Should complete without race condition
}

func TestStepContentStore_ClearBefore(t *testing.T) {
	store := NewStepContentStore()

	// Populate store with several steps
	store.Append(0, "step 0")
	store.Append(1, "step 1")
	store.Append(2, "step 2")
	store.Append(3, "step 3")
	store.Append(4, "step 4")

	// ClearBefore(3) should remove steps 0 and 1 (< 3-1=2), keep 2,3,4
	store.ClearBefore(3)

	if got := store.Get(0); got != "" {
		t.Errorf("ClearBefore: step 0 should be cleared, got %q", got)
	}
	if got := store.Get(1); got != "" {
		t.Errorf("ClearBefore: step 1 should be cleared, got %q", got)
	}
	if got := store.Get(2); got != "step 2" {
		t.Errorf("ClearBefore: step 2 should be kept, got %q", got)
	}
	if got := store.Get(3); got != "step 3" {
		t.Errorf("ClearBefore: step 3 should be kept, got %q", got)
	}
	if got := store.Get(4); got != "step 4" {
		t.Errorf("ClearBefore: step 4 should be kept, got %q", got)
	}

	if store.Count() != 3 {
		t.Errorf("ClearBefore: expected 3 remaining steps, got %d", store.Count())
	}
}

func TestStepContentStore_ClearBefore_EmptyStore(t *testing.T) {
	store := NewStepContentStore()

	// Should not panic on empty store
	store.ClearBefore(5)

	if store.Count() != 0 {
		t.Errorf("ClearBefore empty: expected 0 steps, got %d", store.Count())
	}
}

func TestStepContentStore_ClearBefore_ThreadSafety(t *testing.T) {
	store := NewStepContentStore()

	// Populate store
	for i := 0; i < 100; i++ {
		store.Append(i, "content")
	}

	var wg sync.WaitGroup
	// Concurrent ClearBefore and Append
	wg.Add(200)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			defer wg.Done()
			store.ClearBefore(idx)
		}(i)
		go func(idx int) {
			defer wg.Done()
			store.Append(idx+200, "new content")
		}(i)
	}
	wg.Wait()

	// Should complete without race condition
	_ = store.GetAll()
}

func TestNewStepContentStore(t *testing.T) {
	store := NewStepContentStore()

	if store == nil {
		t.Fatal("NewStepContentStore returned nil")
	}

	if store.content == nil {
		t.Error("NewStepContentStore: content map is nil")
	}

	// Should be empty initially
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("NewStepContentStore: not empty, got %d items", len(all))
	}
}
