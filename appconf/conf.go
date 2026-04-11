package appconf

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"gometeo/mfmap/schedule"
)

const (
	DEFAULT_ADDR = ":1051"

	UPSTREAM_ROOT = "https://meteofrance.com"

	VUE_DEV  = "vue.esm-browser.dev.js"
	VUE_PROD = "vue.esm-browser.prod.js"

	// chorniques history retention
	KEEP_DAY_MIN = -2
	KEEP_DAY_MAX = 2
)

const (
	fastHotDuration = 30 * time.Minute
	fastHotMaxAge   = 1 * time.Minute
	fastColdMaxAge  = 5 * time.Minute

	normalHotDuration = 72 * time.Hour
	normalHotMaxAge   = 60 * time.Minute
	normalColdMaxAge  = 240 * time.Minute
)

type CliOpts struct {
	Addr       string
	Upstream   string
	OneShot    bool
	Limit      int
	Vue        string
	FastUpdate bool
	CacheFile  string
}

var appOpts *CliOpts

var cacheId string

func init() {
	n := uint32(time.Now().UnixMilli() & 0xFFFFFFFF)
	n |= 0x1 << 31 // force left bit to 1 so hex string length is not shorter than 8 chars
	cacheId = fmt.Sprintf("%8x", n)
	slog.Info("cacheId generated", "cacheId", cacheId)
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

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getOpts(args []string) (*CliOpts, error) {
	f := flag.NewFlagSet("Gometeo", flag.ContinueOnError)
	opts := CliOpts{}

	// define cli flags — env vars override hardcoded defaults, explicit flags override env vars
	f.StringVar(&opts.Addr, "addr", envDefault("GOMETEO_ADDR", DEFAULT_ADDR), "listening server address")
	opts.Upstream = envDefault("GOMETEO_UPSTREAM", UPSTREAM_ROOT)
	f.IntVar(&opts.Limit, "limit", 0, "limit number of maps")
	f.BoolVar(&opts.OneShot, "oneshot", false, "useful only for dev and debug")
	f.StringVar(&opts.Vue, "vue", "prod", "select 'prod' or 'dev' build of vue.js")
	f.BoolVar(&opts.FastUpdate, "fastupdate", false, "increase update rate (for dev)")
	f.StringVar(&opts.CacheFile, "cache", "", "path to .gob cache file for oneshot mode (empty = disabled)")

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

func Upstream() string {
	return appOpts.Upstream
}

func OneShot() bool {
	return appOpts.OneShot
}

func Limit() int {
	return appOpts.Limit
}

// VueProd select which vue.js file is called from mail html template
func VueJs() string {
	if appOpts.Vue == "dev" {
		return VUE_DEV
	}
	return VUE_PROD
}

// CacheFile returns the path to the .gob cache file, or "" if disabled.
func CacheFile() string {
	return appOpts.CacheFile
}

func KeepDays() (dayMin, dayMax int) {
	return KEEP_DAY_MIN, KEEP_DAY_MAX
}

func UpdateRate() schedule.UpdateRates {
	if appOpts != nil && appOpts.FastUpdate {
		return schedule.UpdateRates{
			HotDuration: fastHotDuration,
			HotMaxAge:   fastHotMaxAge,
			ColdMaxAge:  fastColdMaxAge,
		}
	}
	return schedule.UpdateRates{
		HotDuration: normalHotDuration,
		HotMaxAge:   normalHotMaxAge,
		ColdMaxAge:  normalColdMaxAge,
	}
}
