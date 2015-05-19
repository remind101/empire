package empire

import (
	"reflect"
	"strings"
	"testing"
	"time"

	gosql "database/sql"

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

// scopeTest is a struct for testing scopes.
type scopeTest struct {
	scope Scope
	sql   string
	vars  []interface{}
}

// scopeTests provides a convenient way to run assertScopeSql on multiple
// scopeTest instances.
type scopeTests []scopeTest

// Run calls assertScopeSql for each scopeTest.
func (tests scopeTests) Run(t testing.TB) {
	for i, tt := range tests {
		sql, vars := conditionSql(tt.scope)

		if got, want := sql, tt.sql; got != want {
			t.Fatalf("#%d: SQL => %v; want %v", i, got, want)
		}

		if got, want := vars, tt.vars; !reflect.DeepEqual(got, want) {
			if len(got) > 0 && len(want) > 0 {
				t.Fatalf("#%d: Vars => %v; want %v", i, got, want)
			}
		}
	}
}

// conditionSql takes a Scope and generates the condition sql that gorm will use
// for the query.
func conditionSql(scope Scope) (sql string, vars []interface{}) {
	db, _ := gorm.Open("postgres", &gosql.DB{})
	ds := scope.Scope(&db).NewScope(nil)
	sql = strings.TrimSpace(ds.CombinedConditionSql())
	vars = ds.SqlVars
	return
}
