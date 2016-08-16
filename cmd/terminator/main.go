package main

import (
	"log"

	"github.com/timehop/docker-cloud-terminator/terminator"
)

func main() {
	// Be kind to devs and include line numbers with each log logsput.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	terminator.Start(terminator.ConfigFromEnv())
}
