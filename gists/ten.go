func main() {
	source := []string{"FOO", "BAR", "BAX"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChannel, err := producer(ctx, source)
	if err != nil {
		log.Fatal(err)
	}

	outputChannel := make(chan string)
	errorChannel := make(chan error)

	limit := int64(runtime.NumCPU())
	sem := semaphore.NewWeighted(limit)

	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case s, ok := <-inputChannel:
				if ok {
					if err := sem.Acquire(ctx, 1); err != nil {
						log.Printf("Failed to acquire semaphore: %v", err)
						break
					}

					go func(s string) {
						defer sem.Release(1)
						time.Sleep(time.Second * 3)

						result := strings.ToLower(s)
						outputChannel <- result
					}(s)
				} else {
					if err := sem.Acquire(ctx, limit); err != nil {
						log.Printf("Failed to acquire semaphore: %v", err)
					}
					close(outputChannel)
					close(errorChannel)
				}
			}
		}

	}()

	sink(ctx, cancel, outputChannel, errorChannel)
}
