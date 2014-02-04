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

const SERVER = "/tmp/gomux.sock"

func main() {
	flHelp := flag.Bool("h", false, "print help text")
	flag.Parse()
	if *flHelp {
		fmt.Println(help)
		os.Exit(0)
		return
	}
	args := flag.Args()
	if len(args) == 0 {
		_, err := os.Stat(SERVER)
		if err == nil {
			fmt.Printf("%s exists. Removing it...\n", SERVER)
			os.RemoveAll(SERVER)
		}
		m := gomux.NewMux(SERVER)
		m.ListenAndServe()
	} else if args[0] == "attach" {
		sessionId, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("Invalid session id: ", args[1])
			os.Exit(1)
			return
		}
		client := gomux.NewClient(os.Stdin, os.Stdout, sessionId)
		client.Connect(SERVER)
	}
}
