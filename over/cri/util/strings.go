package util

import "strings"

// InStringSlice checks whether a string is inside a string slice.
// Comparison is case insensitive.
func InStringSlice(ss []string, str string) bool {
	for _, s := range ss {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}

// SubtractStringSlice subtracts string from string slice.
// Comparison is case insensitive.
func SubtractStringSlice(ss []string, str string) []string {
	var res []string
	for _, s := range ss {
		if strings.EqualFold(s, str) {
			continue
		}
		res = append(res, s)
	}
	return res
}

// MergeStringSlices merges 2 string slices into one and remove duplicated elements.
func MergeStringSlices(a []string, b []string) []string {
	set := map[string]struct{}{}
	for _, s := range a {
		set[s] = struct{}{}
	}
	for _, s := range b {
		set[s] = struct{}{}
	}
	var ss []string
	for s := range set {
		ss = append(ss, s)
	}
	return ss
}
