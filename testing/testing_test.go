package testing

import "fmt"

type exampleTestingT struct{}

func (_ *exampleTestingT) Errorf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func (_ *exampleTestingT) Fatalf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func (_ *exampleTestingT) Helper() {
}

var t = &exampleTestingT{}
