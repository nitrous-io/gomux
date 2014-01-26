package gomux

import (
	"log"
)

func Debugln(a ...interface{}) {
	log.Println(a...)
}

func Debugf(msg string, a ...interface{}) {
	log.Printf(msg, a...)
}
