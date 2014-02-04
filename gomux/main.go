package main

import (
	"flag"
	"fmt"
	"github.com/action-io/gomux"
	"os"
	"strconv"
)

var help = `GOMUX: a pty multiplexer

Gomux has a server mode and client mode. To start a server, simply use "gomux"
command. After the server is started, you can create/attach to a gomux session
using "gomux attach session_id"

- gomux: start gomux server
- gomux attach 1: attach to session 1`

func main() {
	flHelp := flag.Bool("h", false, "print help text")
	flag.Parse()
	if *flHelp {
		fmt.Println(help)
		os.Exit(0)
		return
	}
	args := flag.Args()
	switch args[0] {
	case "attach":
		sessionId, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("Invalid session id: ", args[1])
			os.Exit(1)
			return
		}
		client := gomux.NewClient(os.Stdin, os.Stdout, sessionId)
		client.Connect()
	default:
		m := gomux.NewMux()
		m.ListenAndServe()
	}
}
