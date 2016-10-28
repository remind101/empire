errset is a trivial golang package that implements a slice of errors.

[![Build Status](https://travis-ci.org/crewjam/errset.svg?branch=master)](https://travis-ci.org/crewjam/errset)

[![](https://godoc.org/github.com/crewjam/errset?status.png)](http://godoc.org/github.com/crewjam/errset)

The typical go idiom is to return an `error` or a tuple of (thing, `error`) from functions. This works well if a function performs exactly one task, but
when a function does work which can reasonably partially fail, I found myself writing the same code over and over again. For example:

    // CommitBatch commits as many things as it can.
    func CommitBatch(things []Thing) error {
        errs := errset.ErrSet{}
        for _, thing := range things {
            err := Commit(thing)
            if err != nil {
                errs = append(errs, err)
            }
        }
        return errs.ReturnValue()   // nil if there were no errors
    }

