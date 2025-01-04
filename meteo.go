package main

import (
	"log"
	"os"

	"gometeo/server"
)


func main() {
	err := server.StartSimple(":5151")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
