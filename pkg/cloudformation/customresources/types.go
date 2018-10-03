package customresources

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// IntValue defines an int64 type that can parse integers as strings from json.
// It's common to use `Ref`'s inside templates, which means the value of some
// properties could be a string or an integer.
type IntValue int64

func Int(v int64) *IntValue {
	i := IntValue(v)
	return &i
}

// Eq returns true of other is the same _value_ as i.
func (i *IntValue) Eq(other *IntValue) bool {
	if i == nil {
		return other == nil
	}

	if other == nil {
		return i == nil
	}

	return *i == *other
}

func (i *IntValue) UnmarshalJSON(b []byte) error {
	var si int64
	if err := json.Unmarshal(b, &si); err == nil {
		*i = IntValue(si)
		return nil
	}

	v, err := strconv.Atoi(string(b[1 : len(b)-1]))
	if err != nil {
		return fmt.Errorf("error parsing int from string: %v", err)
	}

	*i = IntValue(v)
	return nil
}

func (i *IntValue) Value() *int64 {
	if i == nil {
		return nil
	}
	p := int64(*i)
	return &p
}
