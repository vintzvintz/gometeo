package appconf

import (
	"flag"
	"fmt"
	"log"
	"time"
)

const (
	DEFAULT_ADDR = ":1051"
)

type CliOpts struct {
	Addr    string
	OneShot bool
	Limit   int
	Vue     string
}

var appOpts *CliOpts

var cacheId string

func init() {
	const magic32bit = 0xdeadbeef
	n := uint32(time.Now().UnixMilli() & 0xFFFFFFFF)
	cacheId = fmt.Sprintf("%8x", n^magic32bit)
}

func CacheId() string {
	if cacheId == "" {
		log.Fatalf("cache id is not initialized")
	}
	return cacheId
}

func Init(args []string) {
	var err error
	appOpts, err = getOpts(args)
	if err != nil {
		//TODO print usage
		log.Fatal(err)
	}
}

func getOpts(args []string) (*CliOpts, error) {
	f := flag.NewFlagSet("Gometeo", flag.ContinueOnError)
	opts := CliOpts{}

	// define cli flags
	f.StringVar(&opts.Addr, "addr", DEFAULT_ADDR, "listening server address")
	f.IntVar(&opts.Limit, "limit", 0, "limit number of maps")
	f.BoolVar(&opts.OneShot, "oneshot", false, "useful only for dev and debug")
	f.StringVar(&opts.Vue, "vue", "prod", "select 'prod' or 'dev' build of vue.js")

	f.Parse(args)

	// validate flag --limit
	if opts.Limit < 0 {
		return nil, fmt.Errorf("invalid cli flag -limit '%d'", opts.Limit)
	}

	// validate flag --vue
	switch opts.Vue {
	case "dev":
		break
	case "prod":
		break
	default:
		return nil, fmt.Errorf("unknown cmdline flag -vue '%s'", appOpts.Vue)
	}
	return &opts, nil
}

func Addr() string {
	return appOpts.Addr
}

func OneShot() bool {
	return appOpts.OneShot
}

func Limit() int {
	return appOpts.Limit
}

// VueProd select which vue.js file is called from mail html template
func VueProd() bool {
	return appOpts.Vue != "dev"
}
