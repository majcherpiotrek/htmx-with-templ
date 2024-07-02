package utils

func ConditionalClassName(base string, condition bool, onTrue string, onFalse string) string {
	if condition && len(onTrue) > 0 {
		return base + " " + onTrue
	}

	if !condition && len(onFalse) > 0 {
		return base + " " + onFalse
	}

	return base
}
