package main

import (
	"context"
	"errors"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"
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

func transformToLower(ctx context.Context, values <-chan string) (<-chan string, <-chan error, error) {
	outChannel := make(chan string)
	errorChannel := make(chan error)

	go func() {
		defer close(outChannel)
		defer close(errorChannel)

		for s := range values {
			time.Sleep(time.Second * 3)
			select {
			case <-ctx.Done():
				return
			default:
				outChannel <- strings.ToLower(s)
			}
		}
	}()

	return outChannel, errorChannel, nil
}

func transformToTitle(ctx context.Context, values <-chan string) (<-chan string, <-chan error, error) {
	outChannel := make(chan string)
	errorChannel := make(chan error)

	go func() {
		defer close(outChannel)
		defer close(errorChannel)

		for s := range values {
			time.Sleep(time.Second * 3)
			select {
			case <-ctx.Done():
				return
			default:
				if s == "foo" {
					errorChannel <- errors.New("error in transformToTitle")
				} else {
					outChannel <- strings.ToTitle(s)
				}
			}
		}
	}()

	return outChannel, errorChannel, nil
}

func sink(ctx context.Context, cancel context.CancelFunc,values <-chan string, errors <-chan error) {
	for {
		select {
		case <-ctx.Done():
			log.Print(ctx.Err().Error())
			return
		case err, ok := <-errors:
			if ok {
       cancel()
				log.Print(err.Error())
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

func main() {
	source := []string{"FOO", "BAR", "BAX"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outputChannel, err := producer(ctx, source)
	if err != nil {
		log.Fatal(err)
	}

	stage1Channels := []<-chan string{}
	errors := []<-chan error{}

	for i := 0; i < runtime.NumCPU(); i++ {
		lowerCaseChannel, lowerCaseErrors, err := transformToLower(ctx, outputChannel)
		if err != nil {
			log.Fatal(err)
		}
		stage1Channels = append(stage1Channels, lowerCaseChannel)
		errors = append(errors, lowerCaseErrors)
	}

	stage1Merged := mergeChans(ctx, stage1Channels...)
	stage2Channels := []<-chan string{}

	for i := 0; i < runtime.NumCPU(); i++ {
		titleCaseChannel, titleCaseErrors, err := transformToTitle(ctx, stage1Merged)
		if err != nil {
			log.Fatal(err)
		}
		stage2Channels = append(stage2Channels, titleCaseChannel)
		errors = append(errors, titleCaseErrors)
	}

	stage2Merged := mergeChans(ctx, stage2Channels...)
	errorsMerged := mergeChans(ctx, errors...)
	sink(ctx, cancel, stage2Merged, errorsMerged)
}

func mergeChans[T any](ctx context.Context, cs ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	out := make(chan T)

	output := func(c <-chan T) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
