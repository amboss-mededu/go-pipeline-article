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
			default:
				outChannel <- s
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

		select {
		case <-ctx.Done():
			return
		case s, ok := <-values:
			if ok {
				log.Println("transformToLower input: ", s)
				time.Sleep(time.Second * 3)
				outChannel <- strings.ToLower(s)
			} else {
				return
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

		select {
		case <-ctx.Done():
			return
		case s, ok := <-values:
			if ok {
				log.Println("transformToTitle input: ", s)
				time.Sleep(time.Second * 3)
				if s == "foo" {
					errorChannel <- errors.New("error in transformToTitle")
				} else {
					outChannel <- strings.ToTitle(s)
				}
			} else {
				return
			}
		}
	}()

	return outChannel, errorChannel, nil
}

func sink(ctx context.Context, cancel context.CancelFunc, values <-chan string, errors <-chan error) {
	for {
		select {
		case <-ctx.Done():
			log.Println(ctx.Err().Error())
			return
		case err, ok := <-errors:
			if ok {
				cancel()
				log.Println(err.Error())
			}
		case val, ok := <-values:
			if ok {
				log.Println("Sink: ", val)
			} else {
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

	stage1Merged := mergeStringChans(ctx, stage1Channels...)
	stage2Channels := []<-chan string{}

	for i := 0; i < runtime.NumCPU(); i++ {
		titleCaseChannel, titleCaseErrors, err := transformToTitle(ctx, stage1Merged)
		if err != nil {
			log.Fatal(err)
		}
		stage2Channels = append(stage2Channels, titleCaseChannel)
		errors = append(errors, titleCaseErrors)
	}

	stage2Merged := mergeStringChans(ctx, stage2Channels...)
	errorsMerged := mergeErrorChans(ctx, errors...)
	sink(ctx, cancel, stage2Merged, errorsMerged)
}

func mergeStringChans(ctx context.Context, cs ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	out := make(chan string)

	output := func(c <-chan string) {
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

func mergeErrorChans(ctx context.Context, cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	output := func(c <-chan error) {
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
