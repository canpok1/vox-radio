package model

// NonNil returns s if s is non-nil, otherwise returns an empty non-nil slice.
// Prevents JSON marshaling from producing null instead of [].
func NonNil[S ~[]E, E any](s S) S {
	if s == nil {
		return make(S, 0)
	}
	return s
}
