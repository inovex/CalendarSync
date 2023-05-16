package sync

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
)

func TestParallelWithSingleTask(t *testing.T) {
	flag := false

	tasks := []taskFunc{
		func() error {
			flag = true
			return nil
		},
	}

	err := parallel(context.TODO(), 1, tasks)
	if err != nil {
		t.Errorf("unexpected error after running parallel: %v", err)
	}
	if !flag {
		t.Error("task was not executed properly")
	}
}

func TestParallelWithManyTasks(t *testing.T) {
	var m sync.Mutex
	ctr := 0

	tasks := []taskFunc{}
	taskCount := 100
	for i := 0; i < taskCount; i++ {
		tasks = append(tasks, func() error {
			m.Lock()
			defer m.Unlock()
			ctr++
			return nil
		})
	}

	err := parallel(context.TODO(), 5, tasks)
	if err != nil {
		t.Errorf("unexpected error after running parallel: %v", err)
	}
	if ctr != taskCount {
		t.Errorf("unexpected number of task invocations, expected %v, got %v", taskCount, ctr)
	}
}

type unwraper interface {
	Unwrap() []error
}

func TestParallelErrorCollection(t *testing.T) {
	errFoo := errors.New("foo")
	errBar := errors.New("bar")
	tasks := []taskFunc{
		func() error {
			return errFoo
		},
		func() error {
			return errBar
		},
	}

	err := parallel(context.TODO(), 5, tasks)
	if err == nil {
		t.Errorf("missing errors")
	}

	errs := err.(unwraper).Unwrap()
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Error() < errs[j].Error()
	})

	expectedErrs := []error{
		errBar,
		errFoo,
	}

	if len(expectedErrs) != len(errs) {
		t.Fatalf("wrong number of errors returned, got %v", errs)
	}
	for i := 0; i < len(expectedErrs); i++ {
		if expectedErrs[i] != errs[i] {
			t.Fatalf("unexpected error at position %v, expected %v, got %v", i, expectedErrs[i], errs[i])
		}
	}
}

func TestParallelCanceledContext(t *testing.T) {
	flag := false

	tasks := []taskFunc{
		func() error {
			flag = true
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	cancel()

	err := parallel(ctx, 1, tasks)
	if err != nil {
		t.Errorf("unexpected error after running parallel: %v", err)
	}
	if flag {
		t.Error("task should not have been executed")
	}
}
