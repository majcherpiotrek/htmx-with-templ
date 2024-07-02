package utils

type ConditionalClass struct {
	Condition bool
	OnTrue    string
	OnFalse   string
}

func NewConditionalClass(condition bool, onTrue string, onFalse string) ConditionalClass {
	return ConditionalClass{
		Condition: condition,
		OnTrue:    onTrue,
		OnFalse:   onFalse,
	}
}

func ClassNames(base string, conditions ...ConditionalClass) string {
	var result = base

	for _, cond := range conditions {
		if cond.Condition && len(cond.OnTrue) > 0 {
			result = result + " " + cond.OnTrue
		}

		if !cond.Condition && len(cond.OnFalse) > 0 {
			result = result + " " + cond.OnFalse
		}
	}

	return result
}
