package main

import (
	"log/slog"
	"os"

	"gometeo/appconf"
	"gometeo/server"
)

func main() {

	appconf.Init(os.Args[1:])

	err := server.Start()
	if err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
