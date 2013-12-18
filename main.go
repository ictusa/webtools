// webtools startup and CLI
//
package main

import (
	"log"
	"os"
	"os/user"
	"strconv"
	// "strings"
)

var DEBUG bool
var DEBUGLVL int
var SchedulerAddress string
var AppID string

// Fetch basic configuration variables.
func init() {
	if val := os.Getenv("WT_DEBUG"); val != "" {
		switch {
		case val == "0":
			DEBUG = false
		case val == "1":
			DEBUG = true
			log.Println("Webtools debugging enabled.")
		}
	}

	if val := os.Getenv("WT_SCHED"); val != "" {
		SchedulerAddress = val
	} else {
		SchedulerAddress = "tcp://localhost:9912"
	}

	if val := os.Getenv("WT_APPID"); val != "" {
		AppID = val
	} else {
		if u, e := user.Current(); e == nil {
			AppID = u.Username
		} else {
			log.Fatalln(e.Error())
		}
	}

	if val := os.Getenv("WT_DEBUGLVL"); val != "" {
		DEBUGLVL, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalln("WT_DEBUGLVL env var must be an integer between 0 and 4", err.Error())
		}

		if DEBUGLVL < 0 || DEBUGLVL > 4 {
			log.Fatalln("WT_DEBUGLVL env var must be an integer between 0 and 4")

		}
		// log.Println("Webtools debug level set to: ", DEBUGLVL)
	} else {
		DEBUGLVL = 0
	}
}

func main() {
	if len(os.Args) > 1 {
		ParseCli(os.Args[1:len(os.Args)])
	} else {
		go SchedulerSigHUPHandler()
		SchedulerService()
	}
}
