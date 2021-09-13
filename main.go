package main

import (
  "errors"
	"context"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
)

func producer(ctx context.Context, strings []string) (<-chan string, error) {
	outChannel := make(chan string)

	go func() {
		defer close(outChannel)

		for _, s := range strings {
			select {
			case <-ctx.Done():
				return
			case outChannel <- s:
			}
		}
	}()

	return outChannel, nil
}

func sink(ctx context.Context, cancelFunc context.CancelFunc, values <-chan string, errors <-chan error) {
	for {
		select {
		case <-ctx.Done():
			log.Print(ctx.Err().Error())
			return
		case err := <-errors:
			if err != nil {
				log.Println("error: ", err.Error())
				cancelFunc()
			}
		case val, ok := <-values:
			if ok {
				log.Printf("sink: %s", val)
			} else {
				log.Print("done")
				return
			}
		}
	}
}

func step[In any,Out any](
  ctx context.Context,
  inputChannel <-chan In,
  outputChannel chan Out,
  errorChannel chan error,
  fn func(In) (Out, error),
) {
	defer close(outputChannel)

	limit := runtime.NumCPU()
	sem1 := semaphore.NewWeighted(limit)

	for s := range inputChannel {
		select {
		case <-ctx.Done():
			log.Print("1 abort")
			break
		default:
		}

		if err := sem1.Acquire(ctx, 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			break
		}

		go func(s In) {
			defer sem1.Release(1)
			time.Sleep(time.Second * 3)

			result, err := fn(s)
			if err != nil {
				errorChannel <- err
			} else {
				outputChannel <- result
			}
		}(s)
	}

	if err := sem1.Acquire(ctx, 8); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}
}

func main() {
	source := []string{"FOO", "BAR", "BAX"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readStream, err := producer(ctx, source)
	if err != nil {
		log.Fatal(err)
	}

	stage1 := make(chan string)
	errorChannel := make(chan error)

	transformA := func(s string) (string, error) {
		return strings.ToLower(s), nil
	}

	go func() {
		step(ctx, readStream, stage1, errorChannel, transformA)
	}()

	stage2 := make(chan string)

	transformB := func(s string) (string, error) {
		if s == "foo" {
			return "", errors.New("oh no")
		}

		return strings.Title(s), nil
	}

	go func() {
		step(ctx, stage1, stage2, errorChannel, transformB)
	}()

	sink(ctx, cancel, stage2, errorChannel)
}
