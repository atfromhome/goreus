package jsonull_test

import (
	"encoding/json"
	"fmt"

	"github.com/atfromhome/goreus/pkg/jsonull"
)

// Example demonstrates basic usage of JsonNull
func Example() {
	type User struct {
		Name  string                   `json:"name"`
		Email jsonull.JsonNull[string] `json:"email,omitempty"`
	}

	// Field with value
	json1 := `{"name":"John","email":"john@example.com"}`
	var user1 User
	json.Unmarshal([]byte(json1), &user1)
	fmt.Println("IsSet:", user1.Email.IsSet())
	fmt.Println("Value:", user1.Email.Value)

	// Field explicitly null
	json2 := `{"name":"Jane","email":null}`
	var user2 User
	json.Unmarshal([]byte(json2), &user2)
	fmt.Println("IsNull:", user2.Email.IsNull())

	// Field not present
	json3 := `{"name":"Bob"}`
	var user3 User
	json.Unmarshal([]byte(json3), &user3)
	fmt.Println("Present:", user3.Email.Present)

	// Output:
	// IsSet: true
	// Value: john@example.com
	// IsNull: true
	// Present: false
}

// ExampleNewJsonNull demonstrates creating a JsonNull with a value
func ExampleNewJsonNull() {
	email := jsonull.NewJsonNull("user@example.com")
	fmt.Println("Present:", email.Present)
	fmt.Println("Valid:", email.Valid)
	fmt.Println("Value:", email.Value)

	// Output:
	// Present: true
	// Valid: true
	// Value: user@example.com
}

// ExampleNewJsonNullNull demonstrates creating a JsonNull representing null
func ExampleNewJsonNullNull() {
	email := jsonull.NewJsonNullNull[string]()
	fmt.Println("Present:", email.Present)
	fmt.Println("Valid:", email.Valid)
	fmt.Println("IsNull:", email.IsNull())

	// Output:
	// Present: true
	// Valid: false
	// IsNull: true
}

// ExampleJsonNullFromPtr demonstrates creating JsonNull from a pointer
func ExampleJsonNullFromPtr() {
	// From nil pointer
	var ptr *string = nil
	email1 := jsonull.JsonNullFromPtr(ptr)
	fmt.Println("From nil pointer - IsNull:", email1.IsNull())

	// From valid pointer
	str := "test@example.com"
	email2 := jsonull.JsonNullFromPtr(&str)
	fmt.Println("From valid pointer - IsSet:", email2.IsSet())
	fmt.Println("From valid pointer - Value:", email2.Value)

	// Output:
	// From nil pointer - IsNull: true
	// From valid pointer - IsSet: true
	// From valid pointer - Value: test@example.com
}

// ExampleJsonNull_IsNull demonstrates the IsNull method
func ExampleJsonNull_IsNull() {
	// Explicitly null value
	nullValue := jsonull.NewJsonNullNull[string]()
	fmt.Println("Null value:", nullValue.IsNull())

	// Valid value
	validValue := jsonull.NewJsonNull("test")
	fmt.Println("Valid value:", validValue.IsNull())

	// Not present
	var notPresent jsonull.JsonNull[string]
	fmt.Println("Not present:", notPresent.IsNull())

	// Output:
	// Null value: true
	// Valid value: false
	// Not present: false
}

// ExampleJsonNull_IsSet demonstrates the IsSet method
func ExampleJsonNull_IsSet() {
	// Valid value
	validValue := jsonull.NewJsonNull("test")
	fmt.Println("Valid value:", validValue.IsSet())

	// Null value
	nullValue := jsonull.NewJsonNullNull[string]()
	fmt.Println("Null value:", nullValue.IsSet())

	// Not present
	var notPresent jsonull.JsonNull[string]
	fmt.Println("Not present:", notPresent.IsSet())

	// Output:
	// Valid value: true
	// Null value: false
	// Not present: false
}

// ExampleJsonNull_Ptr demonstrates the Ptr method
func ExampleJsonNull_Ptr() {
	// Valid value
	validValue := jsonull.NewJsonNull("test")
	ptr := validValue.Ptr()
	if ptr != nil {
		fmt.Println("Valid value pointer:", *ptr)
	}

	// Null value
	nullValue := jsonull.NewJsonNullNull[string]()
	ptr2 := nullValue.Ptr()
	fmt.Println("Null value pointer is nil:", ptr2 == nil)

	// Output:
	// Valid value pointer: test
	// Null value pointer is nil: true
}

// ExampleJsonNull_OrDefault demonstrates the OrDefault method
func ExampleJsonNull_OrDefault() {
	// Valid value
	validValue := jsonull.NewJsonNull("user@example.com")
	email := validValue.OrDefault("default@example.com")
	fmt.Println("Valid value:", email)

	// Null value
	nullValue := jsonull.NewJsonNullNull[string]()
	email2 := nullValue.OrDefault("default@example.com")
	fmt.Println("Null value:", email2)

	// Not present
	var notPresent jsonull.JsonNull[string]
	email3 := notPresent.OrDefault("default@example.com")
	fmt.Println("Not present:", email3)

	// Output:
	// Valid value: user@example.com
	// Null value: default@example.com
	// Not present: default@example.com
}

// ExampleJsonNull_MustGet demonstrates the MustGet method
func ExampleJsonNull_MustGet() {
	// Valid value - safe to use
	validValue := jsonull.NewJsonNull("test")
	value := validValue.MustGet()
	fmt.Println("Value:", value)

	// Output:
	// Value: test
}

// ExampleJsonNull_String demonstrates the String method
func ExampleJsonNull_String() {
	// Valid value
	validValue := jsonull.NewJsonNull("test")
	fmt.Println(validValue.String())

	// Null value
	nullValue := jsonull.NewJsonNullNull[string]()
	fmt.Println(nullValue.String())

	// Not present
	var notPresent jsonull.JsonNull[string]
	fmt.Println(notPresent.String())

	// Output:
	// JsonNull{test}
	// JsonNull{null}
	// JsonNull{not present}
}

// ExampleJsonNull_UnmarshalJSON demonstrates JSON unmarshaling
func ExampleJsonNull_UnmarshalJSON() {
	type Data struct {
		Value jsonull.JsonNull[int] `json:"value,omitempty"`
	}

	// Valid value
	var d1 Data
	json.Unmarshal([]byte(`{"value":42}`), &d1)
	fmt.Println("Valid - IsSet:", d1.Value.IsSet(), "Value:", d1.Value.Value)

	// Null value
	var d2 Data
	json.Unmarshal([]byte(`{"value":null}`), &d2)
	fmt.Println("Null - IsNull:", d2.Value.IsNull())

	// Not present
	var d3 Data
	json.Unmarshal([]byte(`{}`), &d3)
	fmt.Println("Not present - Present:", d3.Value.Present)

	// Output:
	// Valid - IsSet: true Value: 42
	// Null - IsNull: true
	// Not present - Present: false
}

// ExampleJsonNull_MarshalJSON demonstrates JSON marshaling
func ExampleJsonNull_MarshalJSON() {
	type Data struct {
		Value jsonull.JsonNull[string] `json:"value"`
	}

	// Valid value
	d1 := Data{Value: jsonull.NewJsonNull("test")}
	json1, _ := json.Marshal(d1)
	fmt.Println("Valid:", string(json1))

	// Null value
	d2 := Data{Value: jsonull.NewJsonNullNull[string]()}
	json2, _ := json.Marshal(d2)
	fmt.Println("Null:", string(json2))

	// Not present (zero value will marshal to null)
	d3 := Data{}
	json3, _ := json.Marshal(d3)
	fmt.Println("Not present (zero value):", string(json3))

	// Output:
	// Valid: {"value":"test"}
	// Null: {"value":null}
	// Not present (zero value): {"value":null}
}

// Example_patchEndpoint demonstrates using JsonNull for PATCH endpoints
func Example_patchEndpoint() {
	type UpdateUserRequest struct {
		Name  jsonull.JsonNull[string] `json:"name,omitempty"`
		Email jsonull.JsonNull[string] `json:"email,omitempty"`
		Age   jsonull.JsonNull[int]    `json:"age,omitempty"`
	}

	// Scenario 1: Delete email (set to null)
	req1 := `{"email":null}`
	var update1 UpdateUserRequest
	json.Unmarshal([]byte(req1), &update1)
	if update1.Email.IsNull() {
		fmt.Println("Scenario 1: Delete email")
	}

	// Scenario 2: Update name
	req2 := `{"name":"New Name"}`
	var update2 UpdateUserRequest
	json.Unmarshal([]byte(req2), &update2)
	if update2.Name.IsSet() {
		fmt.Println("Scenario 2: Update name to:", update2.Name.Value)
	}
	if !update2.Email.Present {
		fmt.Println("Scenario 2: Don't touch email")
	}

	// Scenario 3: Update multiple fields
	req3 := `{"name":"John","age":30}`
	var update3 UpdateUserRequest
	json.Unmarshal([]byte(req3), &update3)
	if update3.Name.IsSet() {
		fmt.Println("Scenario 3: Update name to:", update3.Name.Value)
	}
	if update3.Age.IsSet() {
		fmt.Println("Scenario 3: Update age to:", update3.Age.Value)
	}

	// Output:
	// Scenario 1: Delete email
	// Scenario 2: Update name to: New Name
	// Scenario 2: Don't touch email
	// Scenario 3: Update name to: John
	// Scenario 3: Update age to: 30
}

// Example_complexTypes demonstrates using JsonNull with complex types
func Example_complexTypes() {
	type Config struct {
		Settings jsonull.JsonNull[map[string]string] `json:"settings,omitempty"`
		Tags     jsonull.JsonNull[[]string]          `json:"tags,omitempty"`
	}

	jsonData := `{
		"settings": {"theme": "dark", "locale": "en"},
		"tags": ["go", "programming"]
	}`

	var config Config
	json.Unmarshal([]byte(jsonData), &config)

	if config.Settings.IsSet() {
		fmt.Println("Theme:", config.Settings.Value["theme"])
		fmt.Println("Locale:", config.Settings.Value["locale"])
	}

	if config.Tags.IsSet() {
		fmt.Println("Tags:", config.Tags.Value)
	}

	// Output:
	// Theme: dark
	// Locale: en
	// Tags: [go programming]
}
