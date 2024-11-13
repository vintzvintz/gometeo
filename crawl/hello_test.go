package crawl

import (
	//"fmt"
	"fmt"
	"testing"
)

type stringAndLen struct {
	s string
	l int
}

func Test_wesh(t *testing.T) {
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

		got := string_len(tc.s)
		want := tc.l

		if got != want {
			t.Errorf("got %d want %d", got, want)
		}
	}
}

func Example_wesh(){
	l := string_len( "wouèch ")
	fmt.Printf( "%d", l)
	// Output: 7
}
