package generic

//go:noinline
func Apply[T interface{ Run(*int) }](value T, target *int) { value.Run(target) }
