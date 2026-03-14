package collector

import "fmt"

// safeCollect runs fn and recovers from any panic, returning it as an error.
// Use this to wrap each individual collector so a single check failure
// cannot crash the entire collection cycle.
func safeCollect(name string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s collector: %v", name, r)
		}
	}()
	return fn()
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// errPtr returns a pointer to the error string, for embedding in report fields.
func errPtr(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
