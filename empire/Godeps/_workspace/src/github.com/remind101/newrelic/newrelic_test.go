package newrelic

import "testing"

func TestNew(t *testing.T) {
	Init("Test App", "<license key>")
}
