package utils

import "github.com/a-h/templ"

func MergeMap[K comparable, T any](base, toMerge *map[K]T) map[K]T {
	result := make(map[K]T)

	for k, v := range *base {
		result[k] = v
	}

	for k, v := range *toMerge {
		result[k] = v
	}

	return result
}

func MergeAttributes(base, toMerge *templ.Attributes) templ.Attributes {
	merged := MergeMap((*map[string]any)(base), (*map[string]any)(toMerge))

	return templ.Attributes(merged)
}
