package starlark

import (
	"sync"
	"testing"

	"go.starlark.net/starlark"
)

func TestThreadPool_GetPut(t *testing.T) {
	pool := NewThreadPool(5)

	// Get a thread
	thread := pool.Get("test1")
	if thread == nil {
		t.Fatal("Get returned nil")
	}
	if thread.Name != "test1" {
		t.Errorf("thread.Name = %q, want \"test1\"", thread.Name)
	}

	// Return it
	pool.Put(thread)
	if pool.Size() != 1 {
		t.Errorf("pool size = %d, want 1", pool.Size())
	}

	// Get it again - should be reused
	thread2 := pool.Get("test2")
	if pool.Size() != 0 {
		t.Errorf("pool size = %d, want 0 after get", pool.Size())
	}
	if thread2.Name != "test2" {
		t.Errorf("thread.Name = %q, want \"test2\"", thread2.Name)
	}
}

func TestThreadPool_MaxSize(t *testing.T) {
	pool := NewThreadPool(2)

	// Create and return 3 threads
	threads := make([]*starlark.Thread, 3)
	for i := 0; i < 3; i++ {
		threads[i] = pool.Get("test")
	}

	for _, thread := range threads {
		pool.Put(thread)
	}

	// Pool should only have 2 threads (max size)
	if pool.Size() != 2 {
		t.Errorf("pool size = %d, want 2 (max size)", pool.Size())
	}
}

func TestThreadPool_DefaultSize(t *testing.T) {
	pool := NewThreadPool(0) // Should use default

	// Should be able to store at least some threads
	for i := 0; i < 5; i++ {
		pool.Put(pool.Get("test"))
	}

	if pool.Size() == 0 {
		t.Error("pool size should not be 0 after puts")
	}
}

func TestThreadPool_Concurrent(t *testing.T) {
	pool := NewThreadPool(10)
	var wg sync.WaitGroup

	// Concurrently get and put threads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			thread := pool.Get("concurrent")
			// Simulate some work
			pool.Put(thread)
		}(i)
	}

	wg.Wait()

	// Pool should have some threads returned
	if pool.Size() > 10 {
		t.Errorf("pool size = %d, should not exceed max of 10", pool.Size())
	}
}

func TestParallelExecutor_Execute(t *testing.T) {
	globals := starlark.StringDict{
		"x": starlark.MakeInt(10),
		"y": starlark.MakeInt(20),
	}

	executor := NewParallelExecutor(5, globals)

	tasks := []EvalTask{
		{Name: "task1", Expr: "x + 1"},
		{Name: "task2", Expr: "y + 2"},
		{Name: "task3", Expr: "x + y"},
	}

	results := executor.Execute(tasks)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check results (order is preserved)
	expected := []int64{11, 22, 30}
	for i, result := range results {
		if result.Error != nil {
			t.Errorf("task %d error: %v", i, result.Error)
			continue
		}
		val, _ := result.Value.(starlark.Int).Int64()
		if val != expected[i] {
			t.Errorf("task %d result = %d, want %d", i, val, expected[i])
		}
	}
}

func TestParallelExecutor_ExecuteWithErrors(t *testing.T) {
	globals := starlark.StringDict{}
	executor := NewParallelExecutor(2, globals)

	tasks := []EvalTask{
		{Name: "valid", Expr: "1 + 1"},
		{Name: "invalid", Expr: "undefined_var"},
	}

	results := executor.Execute(tasks)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First should succeed
	if results[0].Error != nil {
		t.Errorf("task 0 should succeed, got error: %v", results[0].Error)
	}

	// Second should fail
	if results[1].Error == nil {
		t.Error("task 1 should fail with undefined variable")
	}
}
