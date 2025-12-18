package jsonull

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// JsonNull represents a nullable value of type T that can distinguish between:
// - Not present in JSON (Present=false)
// - Present as null (Present=true, Valid=false)
// - Present with value (Present=true, Valid=true)
//
// This is particularly useful for API endpoints that need to differentiate between:
// - A field not being sent in the request (no update)
// - A field explicitly set to null (delete/clear the value)
// - A field set to a specific value (update with new value)
//
// Zero Value Behavior:
// The zero value of JsonNull[T] has Present=false, Valid=false, and Value=zero value of T.
// This represents a value that was not present in the JSON.
//
// Thread Safety:
// JsonNull is not thread-safe. If used concurrently, external synchronization is required.
//
// Example Usage:
//
//	type User struct {
//	    Name  string           `json:"name"`
//	    Email JsonNull[string] `json:"email,omitempty"`
//	    Age   JsonNull[int]    `json:"age,omitempty"`
//	}
//
//	// PATCH /users/:id
//	// Request: {"email": null}
//	// Result: user.Email.IsNull() == true (explicitly clear email)
//
//	// Request: {"name": "John"}
//	// Result: user.Email.Present == false (don't touch email)
//
//	// Request: {"email": "john@example.com"}
//	// Result: user.Email.IsSet() == true (update email)
type JsonNull[T any] struct {
	Value   T
	Valid   bool
	Present bool
}

// NewJsonNull creates a new JsonNull[T] with the given value.
// The returned JsonNull will have Present=true and Valid=true.
//
// Example:
//
//	email := NewJsonNull("user@example.com")
//	// email.Value = "user@example.com"
//	// email.Valid = true
//	// email.Present = true
func NewJsonNull[T any](value T) JsonNull[T] {
	return JsonNull[T]{
		Value:   value,
		Valid:   true,
		Present: true,
	}
}

// NewJsonNullNull creates a new JsonNull[T] representing an explicit null value.
// The returned JsonNull will have Present=true and Valid=false.
//
// Example:
//
//	email := NewJsonNullNull[string]()
//	// email.IsNull() == true
func NewJsonNullNull[T any]() JsonNull[T] {
	return JsonNull[T]{
		Present: true,
		Valid:   false,
	}
}

// JsonNullFromPtr creates JsonNull from pointer.
// If ptr is nil, returns a JsonNull representing null (Present=true, Valid=false).
// Otherwise, returns a JsonNull with the dereferenced value.
//
// Example:
//
//	var ptr *string = nil
//	email := JsonNullFromPtr(ptr)
//	// email.IsNull() == true
//
//	str := "test@example.com"
//	email2 := JsonNullFromPtr(&str)
//	// email2.IsSet() == true
//	// email2.Value == "test@example.com"
func JsonNullFromPtr[T any](ptr *T) JsonNull[T] {
	if ptr == nil {
		return JsonNull[T]{Present: true, Valid: false}
	}

	return JsonNull[T]{Value: *ptr, Valid: true, Present: true}
}

// IsNull returns true if the value is present but null.
// This means the field was explicitly set to null in the JSON.
func (j JsonNull[T]) IsNull() bool {
	return j.Present && !j.Valid
}

// IsSet returns true if the value is present and valid.
// This means the field was set to an actual value in the JSON.
func (j JsonNull[T]) IsSet() bool {
	return j.Present && j.Valid
}

// Ptr returns a pointer to the value if it is valid, otherwise returns nil.
// This is useful when you need to convert JsonNull to a standard pointer type.
//
// Example:
//
//	email := NewJsonNull("user@example.com")
//	ptr := email.Ptr() // *string pointing to "user@example.com"
//
//	nullEmail := NewJsonNullNull[string]()
//	ptr2 := nullEmail.Ptr() // nil
func (j JsonNull[T]) Ptr() *T {
	if !j.Valid {
		return nil
	}
	return &j.Value
}

// OrDefault returns the value if valid, otherwise returns the provided default value.
// This is a safe way to extract a value with a fallback.
//
// Example:
//
//	email := NewJsonNull("user@example.com")
//	result := email.OrDefault("default@example.com") // "user@example.com"
//
//	nullEmail := NewJsonNullNull[string]()
//	result2 := nullEmail.OrDefault("default@example.com") // "default@example.com"
func (j JsonNull[T]) OrDefault(defaultValue T) T {
	if j.Valid {
		return j.Value
	}
	return defaultValue
}

// MustGet returns the value or panics if not valid.
// Use this only when you are certain the value is valid.
//
// Example:
//
//	email := NewJsonNull("user@example.com")
//	value := email.MustGet() // "user@example.com"
//
//	nullEmail := NewJsonNullNull[string]()
//	value2 := nullEmail.MustGet() // panics!
func (j JsonNull[T]) MustGet() T {
	if !j.Valid {
		panic("jsonull: attempted to get value from invalid JsonNull")
	}
	return j.Value
}

// String returns a string representation of the JsonNull value.
// Useful for debugging and logging.
//
// Example:
//
//	email := NewJsonNull("user@example.com")
//	fmt.Println(email.String()) // "JsonNull{user@example.com}"
//
//	nullEmail := NewJsonNullNull[string]()
//	fmt.Println(nullEmail.String()) // "JsonNull{null}"
//
//	var notPresent JsonNull[string]
//	fmt.Println(notPresent.String()) // "JsonNull{not present}"
func (j JsonNull[T]) String() string {
	if !j.Present {
		return "JsonNull{not present}"
	}
	if !j.Valid {
		return "JsonNull{null}"
	}
	return fmt.Sprintf("JsonNull{%v}", j.Value)
}

// UnmarshalJSON unmarshals the JSON data into the value.
// MUST use pointer receiver to modify the struct fields.
//
// This method sets Present=true to indicate the field was in the JSON.
// If the JSON value is null, Valid is set to false.
// If the JSON value is a valid value, Valid is set to true and Value is populated.
// If unmarshaling fails, an error is returned and Valid remains false.
func (j *JsonNull[T]) UnmarshalJSON(data []byte) error {
	j.Present = true
	j.Valid = false

	if bytes.Equal(data, []byte("null")) {
		return nil
	}

	if err := json.Unmarshal(data, &j.Value); err != nil {
		return err
	}

	j.Valid = true
	return nil
}

// MarshalJSON marshals the value into JSON.
// If Valid is false, returns the JSON null value.
// Otherwise, marshals the contained value.
func (j JsonNull[T]) MarshalJSON() ([]byte, error) {
	if !j.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(j.Value)
}
