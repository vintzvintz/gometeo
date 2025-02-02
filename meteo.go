package main

import (
	"log"
	"os"

	"gometeo/server"
	"gometeo/appconf"
)

func main() {

	appconf.Init()

	err := server.Start()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
