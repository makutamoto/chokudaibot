package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jasonlvhit/gocron"
)

var DATABASE_URL = os.Getenv("DATABASE_URL")

func bots() {
	var err error
	err = initChokudai()
	if err != nil {
		log.Println(err)
		return
	}
	defer deinitChokudai()

	gocron.Start()

	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-stop
}

func main() {
	bots()
}
