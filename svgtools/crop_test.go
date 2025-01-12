package svgtools

import (
	"reflect"
	"testing"
)

func TestViewboxToInt(t *testing.T) {
	type testType struct {
		s        string
		want_vb  Viewbox
		want_err bool
	}
	zero := Viewbox{}
	tests := []testType{
		{"", zero, true},         // error
		{"wesh", zero, true},     // error
		{"0", zero, true},        // error
		{"0000", zero, true},     // error
		{"51515151", zero, true}, // error
		{"0 0 0 0 ", zero, true}, // error
		{" 0 0 0 0", zero, true}, // error
		{"0 0 0 0", zero, false},
		{"51 51 51 51", Viewbox{51, 51, 51, 51}, false},
	}
	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			got_vb, err := parseViewbox([]byte(test.s))
			if (err != nil) != test.want_err {
				t.Fatalf("viewboxToInt(%s) (error!=nil)='%s' want '%v'", test.s, err, test.want_err)
			}
			if !reflect.DeepEqual(got_vb, test.want_vb) {
				t.Fatalf("viewboxToInt(%s)=%v want %v", test.s, got_vb, test.want_vb)
			}
		})
	}
}

func TestPxToInt(t *testing.T) {
	type testType struct {
		px       string
		want_n   int
		want_err bool
	}
	tests := []testType{
		{"", 0, true},      // error
		{"wesh", 0, true},  // error
		{"px", 0, true},    // error
		{"51", 0, true},    // error
		{"51 px", 0, true}, // error
		{" 51px", 0, true}, // error
		{"51px ", 0, true}, // error
		{"51px", 51, false},
		{"0px", 0, false},
		{"1px", 1, false},
		{"-0px", 0, false},
		{"-1px", -1, false},
	}
	for _, test := range tests {
		t.Run(test.px, func(t *testing.T) {
			got_n, err := parsePx([]byte(test.px))
			if (err != nil) != test.want_err {
				t.Fatalf("pxToInt(%s) (error!=nil)='%s' want '%v'", test.px, err, test.want_err)
			}
			if got_n != test.want_n {
				t.Fatalf("pxToInt(%s)=%d want %d", test.px, got_n, test.want_n)
			}
		})
	}
}
