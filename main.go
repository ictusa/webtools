// webtools startup and CLI
//
package main

import (
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
	"os/signal"
	"os/user"
	"syscall"
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
	AgentTimeout     int64
}

// config holds the global application configuration
var config Spec

//ServicesRunning determines wether webtools exists after ParseCli is done.
var ServicesRunning bool

// Initialize configuration variables to their default values
func init() {
	uid, uidErr := user.Current()
	if uidErr != nil {
		log.Fatal(uidErr)
	}
	config = Spec{false, 0, "tcp://localhost:9912", uid.Username,
		"/usr/local/etc/webtools/scheduler.json", "tcp://*:9912", "tcp://*:9924", 30}
}

func main() {
	err := envconfig.Process("wt", &config)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		ParseCli(os.Args[1:len(os.Args)])
	} else {
		DoHelp()
	}

	if ServicesRunning {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM)
		<-c //block until we receive SIGTERM
		log.Println("Webtools shutting down - SIGTERM received.")

	}

}
