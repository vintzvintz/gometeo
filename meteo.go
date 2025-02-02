package main

import (
	"flag"
	"log"
	"os"

	"gometeo/server"
)

const (
	DEFAULT_ADDR = ":1051"
)

type CliOpts struct {
	Addr         string
	SimpleMode   bool
	Limit        int
	CacheContent bool
}

func getOpts(args []string) CliOpts {
	f := flag.NewFlagSet("Gometeo", flag.ContinueOnError)
	opts := CliOpts{}

	// define cli flags
	f.StringVar(&opts.Addr, "addr", "", "listening server address")
	f.IntVar(&opts.Limit, "limit", 0, "limit number of maps")
	f.BoolVar(&opts.SimpleMode, "simple", false, "start a server in simple mode")

	f.Parse(args)
	return opts
}

func main() {
	opts := getOpts(os.Args[1:])
	entryPoint := server.Start
	if opts.SimpleMode {
		entryPoint = server.StartSimple
	}
	addr := DEFAULT_ADDR
	if opts.Addr != "" {
		addr = opts.Addr
	}
	limit := 0
	if opts.Limit > 0 {
		limit = opts.Limit
	}
	
	log.Printf(`Starting gometeo : Addr='%s' SimpleMode=%v Limit=%d`,
	addr, opts.SimpleMode, limit)

	err := entryPoint(addr, limit)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
