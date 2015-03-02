This holds integration and functional tests for empire.

### Shared database

These tests share the same postgres database, so they cannot run in parallel. This is prevented by using `empiretest.Run`:

```go
func TestMain(m *testing.M) {
	empiretest.Run(m)
}
```
