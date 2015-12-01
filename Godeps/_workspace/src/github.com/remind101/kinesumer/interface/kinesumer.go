package kinesumeriface

type Kinesumer interface {
	Begin() (int, error)
	End()
	Records() <-chan Record
}
