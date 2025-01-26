package main

import (
	"reflect"
	"testing"
)

func TestFlags(t *testing.T) {

	tests := []struct {
		args []string
		want CliOpts
	}{
		{[]string{}, CliOpts{}},
		{[]string{"-addr", "wesh"}, CliOpts{Addr: "wesh"}},
		{[]string{"-limit", "10"}, CliOpts{Limit: 10}},
		{[]string{"-simple"}, CliOpts{SimpleMode: true}},
//		{[]string{"-no-cache"}, CliOpts{CacheContent: true}},
	}
	// default
	for _, test := range tests {
		opts := getOpts(test.args)
		if !reflect.DeepEqual(opts, test.want) {
			t.Errorf("cli flag got %v want %v", opts, test.want)
		}

	}

}
