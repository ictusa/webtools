// webtools startup and CLI
//
package main

import (
	"os"
)

var DEBUG bool

// Fetch basic configuration variables.
func init() {
	if val := os.Getenv("WT_DEBUG"); val != "" {
		switch {
		case val == "0":
			DEBUG = false
		case val == "1":
			DEBUG = true
		}
	}
}

func main() {
	if len(os.Args) > 1 {
		ParseCli(os.Args[1:len(os.Args)])
	}
}
