package controller

// boolPtr is a test helper that returns a pointer to a bool value.
// This function is used across multiple test files to create bool pointers
// for testing optional boolean fields.
func boolPtr(b bool) *bool {
	return &b
}
