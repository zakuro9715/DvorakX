package testing

import "fmt"

type exampleTestingT struct{}

func (*exampleTestingT) Errorf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}

func (*exampleTestingT) Fatalf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}

func (*exampleTestingT) Helper() {
}

var t = &exampleTestingT{}
