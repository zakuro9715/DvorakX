package diff_test

import "github.com/zakuro9715/DvorakX/testing/diff"

func ExampleFormat() {
	diff.Format("a b c", "a longer c")
}
