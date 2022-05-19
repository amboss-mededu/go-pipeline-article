func step[In any, Out any](
	ctx context.Context,
	inputChannel <-chan In,
	fn func(In) (Out, error),
) (chan Out, chan error) {
	outputChannel := make(chan Out)
	errorChannel := make(chan error)

	limit := int64(2)
	// Use all CPU cores to maximize efficiency. We'll set the limit to 2 so you
	// can see the values being processed in batches of 2 at a time, in parallel
	// limit := int64(runtime.NumCPU())
	sem1 := semaphore.NewWeighted(limit)

	go func() {
		defer close(outputChannel)
		defer close(errorChannel)

		for s := range inputChannel {
			select {
			case <-ctx.Done():
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

		if err := sem1.Acquire(ctx, limit); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
		}
	}()

	return outputChannel, errorChannel
}
