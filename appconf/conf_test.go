package appconf

import (
	"testing"
)

func TestEmpty(t *testing.T) {
	_, err := getOpts([]string{})
	if err != nil {
		t.Error(err)
	}
}

func TestAddr(t *testing.T) {
	want := "wesh"
	opts, err := getOpts([]string{"-addr", want})
	if err != nil {
		t.Error(err)
	}
	got := opts.Addr
	if got != want {
		t.Errorf("cmdline flag --addr got '%s' want '%s'", got, want)
	}
}

func TestLimit(t *testing.T) {
	want := 12
	opts, err := getOpts([]string{"-limit", "12"})
	if err != nil {
		t.Error(err)
	}
	got := opts.Limit
	if got != want {
		t.Errorf("cmdline flag --limit got '%d' want '%d'", got, want)
	}
}

func TestOneShot(t *testing.T) {
	want := true
	opts, err := getOpts([]string{"-oneshot", "1"})
	if err != nil {
		t.Error(err)
	}
	got := opts.OneShot
	if got != want {
		t.Errorf("cmdline flag --onseshot got '%v' want '%v'", got, want)
	}
}

func TestVue(t *testing.T) {
	want := "prod"
	opts, err := getOpts([]string{"-vue", want})
	if err != nil {
		t.Error(err)
	}
	got := opts.Vue
	if got != want {
		t.Errorf("cmdline flag --addr got '%s' want '%s'", got, want)
	}
}

func TestCacheId(t *testing.T) {
	id := CacheId()
	if len(id)!=8 {
		t.Errorf("CacheId() is not a 8-char string : %s", id)
	} 
	t.Logf("cacheId='%s'", id)
}