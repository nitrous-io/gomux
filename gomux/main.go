package main

import (
	"flag"
	"github.com/action-io/gomux"
	"log"
)

func main() {
	flClient := flag.Bool("c", false, "client mode")
	flClientId := flag.Int("i", 0, "server mode")
	flag.Parse()
	if *flClient {
		log.Println("Start client")
		client := &gomux.Client{SessionId: *flClientId}
		client.Connect()
	} else {
		log.Println("Start server")
		m := gomux.NewMux()
		m.Listen()
	}
}
