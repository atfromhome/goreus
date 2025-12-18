# jsonull

`jsonull` is a Go package that provides a generic nullable type for JSON handling with three-state logic. It allows you to distinguish between:
- A field not present in JSON
- A field explicitly set to `null`
- A field with an actual value

This is particularly useful for PATCH endpoints and partial updates where you need to differentiate between "don't update this field" and "set this field to null".

## Features

- ✅ **Three-state logic**: Not present, null, or valid value
- ✅ **Generic implementation**: Works with any type using Go generics
- ✅ **Type-safe**: Compile-time type checking
- ✅ **JSON compatible**: Implements `json.Marshaler` and `json.Unmarshaler`
- ✅ **Comprehensive API**: Helper methods for common operations
- ✅ **Well-tested**: 96.9% test coverage
- ✅ **Zero dependencies**: Only uses Go standard library

## Installation

```bash
go get github.com/atfromhome/goreus/pkg/jsonull
```

## Quick Start

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/atfromhome/goreus/pkg/jsonull"
)

type User struct {
    Name  string               `json:"name"`
    Email jsonull.JsonNull[string] `json:"email,omitempty"`
    Age   jsonull.JsonNull[int]    `json:"age,omitempty"`
}

func main() {
    // Example 1: Field with value
    json1 := `{"name":"John","email":"john@example.com","age":30}`
    var user1 User
    json.Unmarshal([]byte(json1), &user1)
    
    fmt.Println(user1.Email.IsSet())    // true
    fmt.Println(user1.Email.Value)      // "john@example.com"
    
    // Example 2: Field explicitly null
    json2 := `{"name":"Jane","email":null,"age":25}`
    var user2 User
    json.Unmarshal([]byte(json2), &user2)
    
    fmt.Println(user2.Email.IsNull())   // true
    fmt.Println(user2.Email.Present)    // true
    fmt.Println(user2.Email.Valid)      // false
    
    // Example 3: Field not present
    json3 := `{"name":"Bob"}`
    var user3 User
    json.Unmarshal([]byte(json3), &user3)
    
    fmt.Println(user3.Email.Present)    // false
    fmt.Println(user3.Email.IsSet())    // false
    fmt.Println(user3.Email.IsNull())   // false
}
```

## Use Cases

### PATCH Endpoints

Perfect for HTTP PATCH operations where you need to distinguish between:

```go
// PATCH /api/users/123
type UpdateUserRequest struct {
    Name  jsonull.JsonNull[string] `json:"name,omitempty"`
    Email jsonull.JsonNull[string] `json:"email,omitempty"`
    Age   jsonull.JsonNull[int]    `json:"age,omitempty"`
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
    var req UpdateUserRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Request: {"email": null}
    if req.Email.IsNull() {
        // User explicitly wants to delete their email
        deleteUserEmail(userID)
    }
    
    // Request: {"name": "New Name"}
    if req.Name.IsSet() {
        // User wants to update their name
        updateUserName(userID, req.Name.Value)
    }
    
    // Request: {}
    if !req.Age.Present {
        // Age field not sent, don't touch it
        // Do nothing with age
    }
}
```

### Optional Fields in APIs

```go
type Product struct {
    Name        string                  `json:"name"`
    Description jsonull.JsonNull[string]  `json:"description,omitempty"`
    Price       jsonull.JsonNull[float64] `json:"price,omitempty"`
    Stock       jsonull.JsonNull[int]     `json:"stock,omitempty"`
}
```

## API Reference

### Types

```go
type JsonNull[T any] struct {
    Value   T     // The actual value
    Valid   bool  // True if value is valid (not null)
    Present bool  // True if field was present in JSON
}
```

### Constructors

#### `NewJsonNull[T](value T) JsonNull[T]`
Creates a JsonNull with a valid value.

```go
email := jsonull.NewJsonNull("user@example.com")
// email.Present = true
// email.Valid = true
// email.Value = "user@example.com"
```

#### `NewJsonNullNull[T]() JsonNull[T]`
Creates a JsonNull representing an explicit null value.

```go
email := jsonull.NewJsonNullNull[string]()
// email.Present = true
// email.Valid = false
// email.IsNull() = true
```

#### `JsonNullFromPtr[T](ptr *T) JsonNull[T]`
Creates JsonNull from a pointer. Returns null if pointer is nil.

```go
var ptr *string = nil
email := jsonull.JsonNullFromPtr(ptr)
// email.IsNull() = true

str := "test@example.com"
email2 := jsonull.JsonNullFromPtr(&str)
// email2.IsSet() = true
// email2.Value = "test@example.com"
```

### Methods

#### `IsNull() bool`
Returns true if the field was explicitly set to null in JSON.

```go
// JSON: {"email": null}
user.Email.IsNull() // true
```

#### `IsSet() bool`
Returns true if the field has a valid value.

```go
// JSON: {"email": "user@example.com"}
user.Email.IsSet() // true
```

#### `Ptr() *T`
Returns a pointer to the value if valid, otherwise returns nil.

```go
email := jsonull.NewJsonNull("user@example.com")
ptr := email.Ptr() // *string pointing to "user@example.com"

nullEmail := jsonull.NewJsonNullNull[string]()
ptr2 := nullEmail.Ptr() // nil
```

#### `OrDefault(defaultValue T) T`
Returns the value if valid, otherwise returns the provided default.

```go
email := jsonull.NewJsonNull("user@example.com")
result := email.OrDefault("default@example.com") // "user@example.com"

nullEmail := jsonull.NewJsonNullNull[string]()
result2 := nullEmail.OrDefault("default@example.com") // "default@example.com"
```

#### `MustGet() T`
Returns the value or panics if not valid. Use with caution!

```go
email := jsonull.NewJsonNull("user@example.com")
value := email.MustGet() // "user@example.com"

nullEmail := jsonull.NewJsonNullNull[string]()
value2 := nullEmail.MustGet() // panics!
```

#### `String() string`
Returns a string representation for debugging.

```go
email := jsonull.NewJsonNull("user@example.com")
fmt.Println(email.String()) // "JsonNull{user@example.com}"

nullEmail := jsonull.NewJsonNullNull[string]()
fmt.Println(nullEmail.String()) // "JsonNull{null}"

var notPresent jsonull.JsonNull[string]
fmt.Println(notPresent.String()) // "JsonNull{not present}"
```

## Zero Value Behavior

The zero value of `JsonNull[T]` has:
- `Present = false`
- `Valid = false`
- `Value = zero value of T`

This represents a value that was not present in the JSON.

```go
var email jsonull.JsonNull[string]
fmt.Println(email.Present) // false
fmt.Println(email.IsSet())  // false
fmt.Println(email.IsNull()) // false
```

## Thread Safety

⚠️ **JsonNull is not thread-safe.** If you need to use it concurrently, you must provide external synchronization (e.g., using `sync.RWMutex`).

## Comparison with Other Approaches

### vs `*pointer`

```go
// Using pointer
type User struct {
    Email *string `json:"email,omitempty"`
}
// ❌ Cannot distinguish between null and not present
// ❌ nil pointer means both "not sent" and "null"

// Using JsonNull
type User struct {
    Email jsonull.JsonNull[string] `json:"email,omitempty"`
}
// ✅ Clear distinction: Present=false vs IsNull()=true
```

### vs `sql.NullString`

```go
// Using sql.NullString
type User struct {
    Email sql.NullString `json:"email"`
}
// ❌ Cannot distinguish between null and not present
// ❌ Not generic (need different types for each data type)

// Using JsonNull
type User struct {
    Email jsonull.JsonNull[string] `json:"email,omitempty"`
}
// ✅ Three-state logic
// ✅ Works with any type
```

## Performance

Benchmarks on Apple M2:

```
BenchmarkNewJsonNull-8       	1000000000	         0.32 ns/op	       0 B/op	       0 allocs/op
BenchmarkJsonNullFromPtr-8   	1000000000	         0.30 ns/op	       0 B/op	       0 allocs/op
BenchmarkUnmarshalJSON-8     	 1217712	       990.4 ns/op	     752 B/op	      10 allocs/op
BenchmarkMarshalJSON-8       	 2140656	       568.5 ns/op	     208 B/op	       7 allocs/op
```

## Testing

Run tests:

```bash
go test ./pkg/jsonull/
```

Run tests with coverage:

```bash
go test -cover ./pkg/jsonull/
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./pkg/jsonull/
```

## Examples

### Complete CRUD Example

```go
type Article struct {
    ID      int                       `json:"id"`
    Title   string                    `json:"title"`
    Content jsonull.JsonNull[string]  `json:"content,omitempty"`
    Tags    jsonull.JsonNull[[]string] `json:"tags,omitempty"`
    Views   jsonull.JsonNull[int]     `json:"views,omitempty"`
}

// CREATE - all fields have values
func CreateArticle() {
    article := Article{
        ID:      1,
        Title:   "Hello World",
        Content: jsonull.NewJsonNull("Article content here"),
        Tags:    jsonull.NewJsonNull([]string{"go", "programming"}),
        Views:   jsonull.NewJsonNull(100),
    }
    
    data, _ := json.Marshal(article)
    fmt.Println(string(data))
    // {"id":1,"title":"Hello World","content":"Article content here","tags":["go","programming"],"views":100}
}

// UPDATE - partial update with PATCH
func UpdateArticle(data []byte) {
    var article Article
    json.Unmarshal(data, &article)
    
    // Patch: {"content": null}
    if article.Content.IsNull() {
        fmt.Println("Delete content")
    }
    
    // Patch: {"tags": ["updated", "tags"]}
    if article.Tags.IsSet() {
        fmt.Println("Update tags to:", article.Tags.Value)
    }
    
    // Patch: {}
    if !article.Views.Present {
        fmt.Println("Don't touch views")
    }
}
```

### Working with Complex Types

```go
type Config struct {
    Settings jsonull.JsonNull[map[string]interface{}] `json:"settings,omitempty"`
    Features jsonull.JsonNull[[]string]               `json:"features,omitempty"`
    Metadata jsonull.JsonNull[struct {
        Author  string `json:"author"`
        Version string `json:"version"`
    }] `json:"metadata,omitempty"`
}

func ProcessConfig() {
    jsonData := `{
        "settings": {"theme": "dark", "locale": "en"},
        "features": ["feature1", "feature2"],
        "metadata": null
    }`
    
    var config Config
    json.Unmarshal([]byte(jsonData), &config)
    
    if config.Settings.IsSet() {
        fmt.Println("Settings:", config.Settings.Value)
    }
    
    if config.Metadata.IsNull() {
        fmt.Println("Metadata was explicitly set to null")
    }
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This package is part of the [goreus](https://github.com/atfromhome/goreus) project.
