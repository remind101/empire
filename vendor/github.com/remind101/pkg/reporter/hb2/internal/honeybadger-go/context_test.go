package honeybadger

import "testing"

func TestContextUpdate(t *testing.T) {
	c := Context{"foo": "bar"}
	c.Update(Context{"foo": "baz"})
	if c["foo"] != "baz" {
		t.Errorf("Context should update values. expected=%#v actual=%#v", "baz", c["foo"])
	}
}
