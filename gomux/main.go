package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/action-io/gomux"
	"os"
)

var help = `GOMUX: a pty multiplexer

Gomux has a server mode and client mode. To start a server, simply use "gomux"
command. After the server is started, you can create/attach to a gomux session
using "gomux attach session_id"

- gomux: start gomux server
- gomux attach 1: attach to session 1`

var version string

const SERVER = "/tmp/gomux.sock"

func main() {
	flHelp := flag.Bool("h", false, "print help text")
	flVersion := flag.Bool("v", false, "show current version")
	flag.Parse()
	if *flHelp {
		fmt.Println(help)
		os.Exit(0)
		return
	}
	if *flVersion {
		fmt.Println(version)
		return
	}
	args := flag.Args()
	if len(args) == 0 {
		_, err := os.Stat(SERVER)
		if err == nil {
			info, err := gomux.GetInfo(SERVER)
			if err == nil {
				encoder := json.NewEncoder(os.Stdout)
				encoder.Encode(info)
				os.Exit(0)
			}
			fmt.Printf("%s is not running. Removing sock...\n", SERVER)
			os.RemoveAll(SERVER)
		}
		m := gomux.NewMux(SERVER)
		m.ListenAndServe()
	} else if args[0] == "attach" {
		client := gomux.NewClient(os.Stdin, os.Stdout, args[1])
		err := client.Connect(SERVER)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	} else if args[0] == "events" {
		err := gomux.GetEvents(SERVER)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}
