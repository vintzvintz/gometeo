package crawl

import (
	"fmt"
	"strings"
	"testing"
)

type stringAndLen struct {
	s string
	l int
}

func stringLen(s string) int {
	var l int = strings.Count(s, "")
	return l - 1 // https://pkg.go.dev/strings#example-Count
}

func Test_StringLen(t *testing.T) {
	var test_cases = make([]stringAndLen, 0)

	test_cases = append(test_cases,
		stringAndLen{}, // length of empty string is 0
		stringAndLen{s: "", l: 0},
		stringAndLen{s: "wesh", l: 4},
		stringAndLen{s: "îè世界", l: 4},
	)

	for _, tc := range test_cases {
		//log := fmt.Sprintf("Test case %d", i)
		//t.Log(log)

		got := stringLen(tc.s)
		want := tc.l

		if got != want {
			t.Errorf("got %d want %d", got, want)
		}
	}
}

func ExamplestringLen() {
	l := stringLen("wouèch ")
	fmt.Printf("%d", l)
	// Output: 7
}
