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
	for _, tt := range tests {
		assertScopeSql(t, tt.scope, tt.sql, tt.vars...)
	}
}

// assertScopeSql ensures that the generating sql matches what is expected.
func assertScopeSql(t testing.TB, scope Scope, sql string, vars ...interface{}) {
	db, err := gorm.Open("postgres", &gosql.DB{})
	if err != nil {
		t.Fatal(err)
	}

	ds := scope.Scope(&db).NewScope(nil)
	got := strings.TrimSpace(ds.CombinedConditionSql())

	t.Logf("SQL: %s", got)
	t.Logf("Variables: %v", ds.SqlVars)

	if got != sql {
		t.Fatalf("SQL => %v; want %v", got, sql)
	}

	if got, want := ds.SqlVars, vars; !reflect.DeepEqual(got, want) {
		if len(got) > 0 && len(want) > 0 {
			t.Fatalf("Vars => %v; want %v", got, want)
		}
	}
}
