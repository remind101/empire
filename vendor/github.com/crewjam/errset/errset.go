package errset

// ErrSet represents a list of errors.
//
// Example:
//
//     errs := ErrSet{}
//     for ... {
//       if err := DoSomething(i); err != nil {
//         errs = append(errs, err)
//       }
//     }
//     return errs.ReturnValue()
//
type ErrSet []error

// ReturnValue returns the ErrSet object if at least one non-nill error is
// present or nil if there are no errors
func (es ErrSet) ReturnValue() error {
	rv := ErrSet{}
	for _, err := range es {
		if err != nil {
			rv = append(rv, err)
		}
	}
	if len(rv) == 0 {
		return nil
	}
	return rv
}

// Error implements the error interface. It returns each error in the list
// concatenated together with "; ".
func (es ErrSet) Error() string {
	rv := ""
	errCount := 0
	for _, err := range es {
		if err == nil {
			continue
		}
		if errCount != 0 {
			rv += "; "
		}
		rv += err.Error()
		errCount++
	}
	return rv
}
