package main

import (
	"log"
	"os"

	"gometeo/server"
)

func main() {
	err := server.Start(":5151", 10)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
