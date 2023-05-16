package sync

import (
	"context"
	"errors"
	"sync"
)

type taskFunc func() error

func parallel(ctx context.Context, workerCount int, tasks []taskFunc) error {
	var errs []error
	var errorWg sync.WaitGroup
	var workerWg sync.WaitGroup
	taskCh := make(chan func() error)
	errCh := make(chan error)

	errorWg.Add(1)
	workerWg.Add(workerCount)

	// collect errors
	go func() {
		defer errorWg.Done()
		for err := range errCh {
			errs = append(errs, err)
		}
	}()

	// execute tasks in parallel
	for i := 0; i < workerCount; i++ {
		go func() {
			defer workerWg.Done()
			for task := range taskCh {
				err := task()
				if err != nil {
					select {
					case errCh <- err:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	for _, task := range tasks {
		select {
		case taskCh <- task:
		case <-ctx.Done():
		}
	}

	close(taskCh)
	workerWg.Wait()

	// Close errCh after all worker have stopped
	close(errCh)
	errorWg.Wait()

	return errors.Join(errs...)
}
