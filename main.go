package main

import (
	"log"
)

func producer(strings []string) (<-chan string, error) {
	outChannel := make(chan string)

	for _, s := range strings {
		outChannel <- s
	}

	return outChannel, nil
}

func sink(values <-chan string) {
	for value := range values {
		log.Println(value)
	}
}

func main() {
	source := []string{"foo", "bar", "bax"}

	outputChannel, err := producer(source)
	if err != nil {
		log.Fatal(err)
	}

	sink(outputChannel)
}
