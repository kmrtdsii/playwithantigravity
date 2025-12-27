# GitGym Go Style Guide

GitGym adopts the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) as its primary coding standard. This document highlights key patterns we strictly enforce.

## 1. Interface Compliance
Verify interface compliance at compile time. This ensures that if the interface changes, the compiler catches missing methods immediately.

**Pattern:**
```go
type MyCommand struct{}

// Ensure MyCommand implements git.Command
var _ git.Command = (*MyCommand)(nil)
```

**Apply to:**
- All `Command` implementations in `internal/git/commands`.
- All `Session` or `Storer` implementations.

## 2. Error Handling
- **Wrap Errors**: Use `%w` when wrapping errors to allow callers to unwrap them using `errors.Is` or `errors.As`.
    ```go
    // Bad
    return fmt.Errorf("failed to call foo: %v", err)
    // Good
    return fmt.Errorf("failed to call foo: %w", err)
    ```
- **Static vs Dynamic**:
    - Static strings: `errors.New("something went wrong")`
    - Dynamic strings: `fmt.Errorf("file %s not found", name)`

## 3. Performance
- **Pre-allocate Slices/Maps**: When the size is known or estimated, use `make` with capacity.
    ```go
    // Bad
    var data []string
    for _, item := range items {
        data = append(data, item)
    }

    // Good
    data := make([]string, 0, len(items))
    ```
- **Strconv vs Fmt**: Use `strconv.Itoa(i)` instead of `fmt.Sprintf("%d", i)` for simple conversions.

## 4. Zero Values
- Use zero-value initialization where possible.
- **Mutexes**: `sync.Mutex` and `sync.RWMutex` are valid zero values. No need to point to them.
- **Slices**: `nil` slices are valid.
    ```go
    // Bad
    s := []string{}
    // Good
    var s []string
    ```
