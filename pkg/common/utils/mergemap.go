package utils

import "github.com/a-h/templ"

func MergeMap[K comparable, T any](maps ...*map[K]T) map[K]T {
	result := make(map[K]T)

	for _, toMerge := range maps {
		for k, v := range *toMerge {
			result[k] = v
		}
	}

	return result
}

func MergeAttributes(attrs ...*templ.Attributes) templ.Attributes {
	attrsAsMaps := make([]*map[string]any, len(attrs))

	for i, toMerge := range attrs {
		attrsAsMaps[i] = (*map[string]any)(toMerge)
	}

	merged := MergeMap(attrsAsMaps...)

	return templ.Attributes(merged)
}
