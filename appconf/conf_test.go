package appconf

import (
	"os"
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

func TestAddrEnvVar(t *testing.T) {
	os.Setenv("GOMETEO_ADDR", ":9999")
	defer os.Unsetenv("GOMETEO_ADDR")

	opts, err := getOpts([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Addr != ":9999" {
		t.Errorf("env GOMETEO_ADDR: got %q, want %q", opts.Addr, ":9999")
	}
}

func TestAddrFlagOverridesEnv(t *testing.T) {
	os.Setenv("GOMETEO_ADDR", ":9999")
	defer os.Unsetenv("GOMETEO_ADDR")

	opts, err := getOpts([]string{"-addr", ":8080"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Addr != ":8080" {
		t.Errorf("flag should override env: got %q, want %q", opts.Addr, ":8080")
	}
}

func TestUpstreamEnvVar(t *testing.T) {
	os.Setenv("GOMETEO_UPSTREAM", "https://example.com")
	defer os.Unsetenv("GOMETEO_UPSTREAM")

	opts, err := getOpts([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Upstream != "https://example.com" {
		t.Errorf("env GOMETEO_UPSTREAM: got %q, want %q", opts.Upstream, "https://example.com")
	}
}

func TestUpstreamDefault(t *testing.T) {
	os.Unsetenv("GOMETEO_UPSTREAM")

	opts, err := getOpts([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Upstream != UPSTREAM_ROOT {
		t.Errorf("default upstream: got %q, want %q", opts.Upstream, UPSTREAM_ROOT)
	}
}

func TestCacheId(t *testing.T) {
	id := CacheId()
	if len(id) != 8 {
		t.Errorf("CacheId() is not a 8-char string : %s", id)
	}
	t.Logf("cacheId='%s'", id)
}
