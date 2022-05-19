func main() {
	source := []string{"FOO", "BAR", "BAX"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readStream, err := producer(ctx, source)
	if err != nil {
		log.Fatal(err)
	}

	step1results, step1errors := step(ctx, readStream, transformA)
	step2results, step2errors := step(ctx, step1results, transformB)
	allErrors := Merge(ctx, step1errors, step2errors)

	sink(ctx, cancel, step2results, allErrors)
}
