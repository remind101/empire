package empire

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestComposedScope(t *testing.T) {
	var scope ComposedScope

	a, b := make(chan struct{}), make(chan struct{})

	scope = append(scope, MockScope(a))
	scope = append(scope, MockScope(b))

	db := &gorm.DB{}

	go scope.Scope(db)

	select {
	case <-a:
	case <-time.After(time.Second):
		t.Fatal("Expected a to be called")
	}

	select {
	case <-b:
	default:
		t.Fatal("Expected b to be called")
	}
}

// MockScope is a Scope implementation that closes the channel when it is
// called.
func MockScope(called chan struct{}) Scope {
	return ScopeFunc(func(db *gorm.DB) *gorm.DB {
		close(called)
		return db
	})
}
