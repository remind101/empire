package errset

import (
	"fmt"

	. "gopkg.in/check.v1"
)

type ErrSetTest struct {
}

var _ = Suite(&ErrSetTest{})

func (testSuite *ErrSetTest) TestBasics(c *C) {

	es := ErrSet{}
	c.Assert(es.ReturnValue(), IsNil)
	c.Assert(es.Error(), Equals, "")

	es = append(es, nil)
	c.Assert(es.ReturnValue(), IsNil)
	c.Assert(es.Error(), Equals, "")

	es = append(es, fmt.Errorf("foo"))
	c.Assert(es.ReturnValue(), Not(IsNil))
	c.Assert(es.Error(), Equals, "foo")

	es = append(es, nil)
	c.Assert(es.Error(), Equals, "foo")

	es = append(es, fmt.Errorf("bar"))
	c.Assert(es.ReturnValue(), Not(IsNil))
	c.Assert(es.Error(), Equals, "foo; bar")
}
