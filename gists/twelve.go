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
