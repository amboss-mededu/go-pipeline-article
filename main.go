package main

import (
	"context"
	"log"
	"time"
)

func producer(ctx context.Context, strings []string) (<-chan string, error) {
	outChannel := make(chan string)

	go func() {
		defer close(outChannel)

		for _, s := range strings {
			time.Sleep(time.Second * 3)
			select {
			case <-ctx.Done():
				return
			case outChannel <- s:
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
			log.Print(val)
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
	source := []string{"foo", "bar", "bax"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// go func() {
	// 	time.Sleep(time.Second * 5)
	// 	cancel()
	// }()

	outputChannel, err := producer(ctx, source)
	if err != nil {
		log.Fatal(err)
	}

	sink(ctx, outputChannel)
}
