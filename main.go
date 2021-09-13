package main

import (
	"context"
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

func transformToLower(ctx context.Context, values <-chan string) (<-chan string, error) {
	outChannel := make(chan string)

	go func() {
		defer close(outChannel)

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

	return outChannel, nil
}

func transformToTitle(ctx context.Context, values <-chan string) (<-chan string, error) {
	outChannel := make(chan string)

	go func() {
		defer close(outChannel)

		for s := range values {
			time.Sleep(time.Second * 3)
			select {
			case <-ctx.Done():
				return
			default:
				outChannel <- strings.ToTitle(s)
			}
		}
	}()

	return outChannel, nil
}

func sink(ctx context.Context, values <-chan string) {
	for {
		select {
		case <-ctx.Done():
			log.Print(ctx.Err().Error())
			return
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

	for i := 0; i < runtime.NumCPU(); i++ {
		lowerCaseChannel, err := transformToLower(ctx, outputChannel)
		if err != nil {
			log.Fatal(err)
		}
		stage1Channels = append(stage1Channels, lowerCaseChannel)
	}

	stage1Merged := mergeStringChans(ctx, stage1Channels...)
	stage2Channels := []<-chan string{}

	for i := 0; i < runtime.NumCPU(); i++ {
		titleCaseChannel, err := transformToTitle(ctx, stage1Merged)
		if err != nil {
			log.Fatal(err)
		}
		stage2Channels = append(stage2Channels, titleCaseChannel)
	}

	stage2Merged := mergeStringChans(ctx, stage2Channels...)
	sink(ctx, stage2Merged)
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
