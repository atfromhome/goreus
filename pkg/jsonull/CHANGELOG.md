# Changelog

All notable changes to the `jsonull` package will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-XX

### Added
- Initial release of `jsonull` package
- `JsonNull[T]` generic type with three-state logic (not present, null, or value)
- `NewJsonNull[T](value T)` constructor for creating JsonNull with a value
- `NewJsonNullNull[T]()` constructor for creating JsonNull representing null
- `JsonNullFromPtr[T](ptr *T)` constructor for creating JsonNull from pointer
- `IsNull()` method to check if value is explicitly null
- `IsSet()` method to check if value is present and valid
- `Ptr()` method to get pointer to value
- `OrDefault(defaultValue T)` method to get value or default
- `MustGet()` method to get value or panic
- `String()` method for debugging representation
- `UnmarshalJSON()` implementation for JSON unmarshaling
- `MarshalJSON()` implementation for JSON marshaling
- Comprehensive test suite with 96.9% coverage
- Benchmark tests for performance measurement
- Complete documentation with examples
- README.md with usage guide and API reference
- Example tests for godoc

### Fixed
- Error handling in `UnmarshalJSON` to properly reset `Valid` flag before unmarshaling
- Proper state management to distinguish between unmarshal errors and null values

### Documentation
- Added comprehensive inline documentation for all types and methods
- Added usage examples for common scenarios (PATCH endpoints, optional fields)
- Added comparison with other approaches (pointers, sql.NullString)
- Added zero value behavior documentation
- Added thread safety notes
- Added performance benchmarks

### Testing
- Unit tests for all constructors and methods
- Integration tests for JSON marshal/unmarshal
- Round-trip tests to verify data integrity
- Panic recovery tests for `MustGet()`
- Benchmark tests for performance profiling
- Example tests for godoc

## [Unreleased]

### Planned
- Support for custom JSON null values (e.g., empty string, specific sentinel values)
- Optional validation functions
- Integration with popular validation libraries
- Helper methods for slice operations
- Comparison operators for comparable types

---

## Version History

- **v1.0.0**: Initial stable release with core functionality