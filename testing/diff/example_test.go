package diff_test

import "github.com/zakuro9715/DvorakZ/testing/diff"

func ExampleFormat() {
	diff.Format("a b c", "a longer c")
}
