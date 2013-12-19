// webtools startup and CLI
//
package main

import (
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
	"os/user"
)

// Spec represents the webtools configuration via environment variables
type Spec struct {
	Debug            bool
	DebugLvl         int
	SchedulerAddress string
	AppId            string
	SchedulerDbPath  string
	SchedulerListen  string
	AgentListen      string
}

// config holds the global application configuration
var config Spec

// Initialize configuration variables to their default values
func init() {
	uid, uidErr := user.Current()
	if uidErr != nil {
		log.Fatal(uidErr)
	}
	config = Spec{false, 0, "tcp://localhost:9912", uid.Username,
		"/usr/local/etc/webtools/scheduler.json", "tcp://*:9912", "tcp://*:9924"}
}

func main() {
	err := envconfig.Process("wt", &config)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		ParseCli(os.Args[1:len(os.Args)])
	} else {
		go SchedulerSigHUPHandler()
		SchedulerService()
	}
}
