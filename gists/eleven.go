func step[In any,Out any](
  ctx context.Context,
  inputChannel <-chan In,
  outputChannel chan Out,
  errorChannel chan error,
  fn func(In) (Out, error),
) {
	defer close(outputChannel)

	sem1 := semaphore.NewWeighted(8)

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