package main

import (
	"log"
	"os"

	"gometeo/crawl"
	"gometeo/server"
)

func main() {

	crawler := crawl.NewCrawler()

	m, err := crawler.GetMap("/", nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = server.StartSimple(server.MapCollection{m})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
