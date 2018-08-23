package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathJoin(t *testing.T) {
	path := PathJoin("a", "b")
	assert.Equal(t, "a/b", path)

	path = PathJoin("a", "b", "c")
	assert.Equal(t, "a/b/c", path)

	path = PathJoin("a", "../b")
	assert.Equal(t, "a/..%2Fb", path)

	path = PathJoin("a", "b/c")
	assert.Equal(t, "a/b%2Fc", path)
}
