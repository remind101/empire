package kinesumeriface

type Kinesumer interface {
	Begin() (err error)
	End()
	Records() <-chan *Record
}
