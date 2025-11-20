package contextforge

import "time"

// Pointer helper functions for working with optional fields.
//
// These helpers simplify creating and dereferencing pointers, which are used
// throughout the SDK to distinguish between three states for optional fields:
//   - nil: field not set (will be omitted from JSON with omitempty tag)
//   - pointer to zero value: field explicitly set to empty/zero
//   - pointer to value: field set to that value
//
// Example usage:
//
//	// Update only the name, leave other fields unchanged
//	update := &contextforge.ResourceUpdate{
//	    Name: contextforge.String("new-name"),
//	    // Description, Tags, etc. are nil and won't be sent
//	}
//
//	// Clear the description (set to empty string)
//	update := &contextforge.ResourceUpdate{
//	    Description: contextforge.String(""),
//	}
//
//	// Don't update tags vs clear all tags
//	update1 := &contextforge.ResourceUpdate{
//	    Tags: nil, // Don't update tags
//	}
//	update2 := &contextforge.ResourceUpdate{
//	    Tags: []string{}, // Clear all tags
//	}

// String returns a pointer to the provided string value.
func String(v string) *string {
	return &v
}

// StringValue returns the value of the string pointer passed in or
// "" if the pointer is nil.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// Int returns a pointer to the provided int value.
func Int(v int) *int {
	return &v
}

// IntValue returns the value of the int pointer passed in or
// 0 if the pointer is nil.
func IntValue(v *int) int {
	if v != nil {
		return *v
	}
	return 0
}

// Bool returns a pointer to the provided bool value.
func Bool(v bool) *bool {
	return &v
}

// BoolValue returns the value of the bool pointer passed in or
// false if the pointer is nil.
func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// Time returns a pointer to the provided time.Time value.
func Time(v time.Time) *time.Time {
	return &v
}

// TimeValue returns the value of the time.Time pointer passed in or
// the zero time if the pointer is nil.
func TimeValue(v *time.Time) time.Time {
	if v != nil {
		return *v
	}
	return time.Time{}
}

// Int64 returns a pointer to the provided int64 value.
func Int64(v int64) *int64 {
	return &v
}

// Int64Value returns the value of the int64 pointer passed in or
// 0 if the pointer is nil.
func Int64Value(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// Float64 returns a pointer to the provided float64 value.
func Float64(v float64) *float64 {
	return &v
}

// Float64Value returns the value of the float64 pointer passed in or
// 0.0 if the pointer is nil.
func Float64Value(v *float64) float64 {
	if v != nil {
		return *v
	}
	return 0.0
}
