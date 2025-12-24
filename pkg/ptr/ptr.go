package ptr

func Of[T any](t T) *T {
	return &t
}

func ValueOr[T any](t *T, or T) T {
	if nil == t {
		return or
	}

	return *t
}
