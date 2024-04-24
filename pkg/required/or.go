package required

func Or[T any](required T, defaultVal T) T {
	if required != nil {
		return required
	}
	return defaultVal
}
