package crawl

import (
	"math"
	"strings"
	"testing"
)


var repCnt int = int(math.Pow( 2, 8) + 1 )

// cas-test pour la conversion ROT-13
var rot13tests = map[string]struct {
	in  string
	out string
	len int
}{
	"empy string": {
		in:  "",
		out: "",
		len: 0,
	},
	"alpha #1": {
		in:  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
		out: "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm",
		len: 52,
	},
	"alpha #2": {
		in:  "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm",
		out: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
		len: 52,
	},
	"digits": {
		in:  "1234567890",
		out: "1234567890",
		len: 10,
	},
	"speciaux": {
		in:  "&#'{([-|è_@°\t\n",
		out: "&#'{([-|è_@°\t\n",
		len: 16,
	},
	"UTF-8": {
		in:  "àéèïÏ",
		out: "àéèïÏ",
		len: 10,
	},
	"mfsession token": {
		in:  `rlWwoTSmplV6VzyhqTIlozI0VvjvLJkaVwbvFSZlAGLvYPW0rKNvBvWXI1DvsD.rlWdqTxvBvWwA2EuA2R2BGV2AmWuLGV2BJEyZmOvAQt1MzV2Amp1ZFVfVzyuqPV6ZGpmZGZmBGp4ZU0.k8paACSN0MkGEp5hwwwXWoqExfP2L8nTxYq49LgJ1tD`,
		out: `eyJjbGFzcyI6ImludGVybmV0IiwiYWxnIjoiSFMyNTYiLCJ0eXAiOiJKV1QifQ.eyJqdGkiOiJjN2RhN2E2OTI2NzJhYTI2OWRlMzBiNDg1ZmI2Nzc1MSIsImlhdCI6MTczMTMzOTc4MH0.x8cnNPFA0ZxTRc5ujjjKJbdRksC2Y8aGkLd49YtW1gQ`,
		len: 186,
	},
	"exemple": {
		in:  `"Lbh penpxrq gur pbqr!"`,
		out: `"You cracked the code!"`,
		len: 23,
	},
	"1M zéros": {
		in:  strings.Repeat("Abcde", repCnt),
		out: strings.Repeat("Nopqr", repCnt),
		len: repCnt*len("Abcde"),
	},
}

// test de la conversion ROT13
func TestRot13(t *testing.T) {
	for name, test := range rot13tests {
		t.Run(name, func(t *testing.T) {
			//t.Parallel()
			got := Rot13(test.in)
			if len(got) != test.len {
				t.Fatalf("rot13(%q) is %d bytes; expected %d bytes", test.in, len(got), test.len)
			}
			if got != test.out {
				t.Fatalf("rot13(%q) returned %q; expected %q", test.in, got, test.out)
			}
		})
	}
}
