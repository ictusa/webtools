//webtools cli parser
package main

import (
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"log"
	"strconv"
)

const (
	CLI_INIT = iota
	CLI_HELP
	CLI_KILL
	CLI_PING
	CLI_PINGSCHED
	CLI_PINGAGENT
	CLI_PS
	CLI_SCHEDULER
	CLI_SCHEDLOOKUP
	CLI_START
	CLI_STOP
	CLI_VERSION
)

// ParseCli implements a very naive parser for command line arguments.
func ParseCli(cmds []string) {
	if DEBUG {
		log.Println("ParseCli(", cmds, ")")
	}
	parsecli(CLI_INIT, cmds)

}

// parsecli is the recursive portion of the parser, should be called by ParseCli only!
func parsecli(state int, cmds []string) {
	var statemap = map[int]string{
		CLI_INIT:        "CLI_INIT",
		CLI_HELP:        "CLI_HELP",
		CLI_KILL:        "CLI_KILL",
		CLI_PING:        "CLI_PING",
		CLI_PINGSCHED:   "CLI_PINGSCHED",
		CLI_PINGAGENT:   "CLI_PINGAGENT",
		CLI_PS:          "CLI_PS",
		CLI_SCHEDULER:   "CLI_SCHEDULER",
		CLI_SCHEDLOOKUP: "CLI_SCHEDLOOKUP",
		CLI_START:       "CLI_START",
		CLI_STOP:        "CLI_STOP",
		CLI_VERSION:     "CLI_VERSION",
	}
	if DEBUG {
		log.Println("parsecli(", statemap[state], ",", cmds, ")")
	}
	switch {
	case state == CLI_INIT:

		switch {
		case cmds[0] == "version":
			parsecli(CLI_VERSION, cmds[1:len(cmds)])

		case cmds[0] == "ping":
			parsecli(CLI_PING, cmds[1:len(cmds)])

		case cmds[0] == "kill":
			parsecli(CLI_KILL, cmds[1:len(cmds)])

		case cmds[0] == "help":
			parsecli(CLI_HELP, cmds[1:len(cmds)])

		case cmds[0] == "scheduler":
			parsecli(CLI_SCHEDULER, cmds[1:len(cmds)])

		case cmds[0] == "start":
			parsecli(CLI_START, cmds[1:len(cmds)])

		case cmds[0] == "stop":
			parsecli(CLI_STOP, cmds[1:len(cmds)])

		case cmds[0] == "ps":
			parsecli(CLI_PS, cmds[1:len(cmds)])

		default:
			fmt.Println("webtools: unknown command:", cmds[0])
			fmt.Println("Run 'webtools help' for usage information.")
		} // state == CLI_INIT

	case state == CLI_VERSION:
		DoVersion()

	case state == CLI_PING:
		switch {
		case cmds[0] == "agent":
			parsecli(CLI_PINGAGENT, cmds[1:len(cmds)])
		case cmds[0] == "scheduler":
			parsecli(CLI_PINGSCHED, cmds[1:len(cmds)])

		} // state == CLI_PING

	case state == CLI_PINGAGENT:
		if len(cmds) != 1 {
			log.Fatalln("Usage: webtools ping agent <hostname>")
		}
		DoPingAgent(cmds[0])
	case state == CLI_PINGSCHED:
		DoPingSched()

	case state == CLI_KILL:
		if len(cmds) != 1 {
			log.Fatalln("Usage: webtools kill <pid>")
		}

		if pid, err := strconv.Atoi(cmds[0]); err == nil {
			DoKill(pid)
		} else {
			log.Fatalln("Usage: webtools kill <pid>, ", err.Error())
		}

	case state == CLI_SCHEDULER:
		if len(cmds) < 1 {
			DoHelp()
		}
		switch {
		case cmds[0] == "lookup":
			if len(cmds) == 2 {
				DoSchedLookup(cmds[1])
			} else {
				DoSchedLookup(AppID)
			}
		default:
			DoHelp()
		}
	case state == CLI_HELP:
		DoHelp()

	case state == CLI_PS:
		DoPs()

	case state == CLI_START:
		DoStart()

	case state == CLI_STOP:
		DoStop()

	} //state switch
}

func DoHelp() {
	fmt.Print("Webtools is an automation tool for use by developers to run commands remotely on content servers.\n" +
		"\n" +
		"Usage:\n" +
		"webtools command <required arguments> [optional arguments] \n" +
		"\n" +
		"The commands are:\n" +
		"  help                     - Display this text\n" +
		"  kill <pid>               - Kill PID on content server under this account\n" +
		"  ping scheduler           - Display status of scheduler\n" +
		"  ping agent <host>        - Display status of agent on host\n" +
		"  ps                       - Display processes on content server under this account\n" +
		"  scheduler lookup [Appid] - Query scheduler for agent address of App\n" +
		"  start                    - Execute ~/bin/start on content server under this account\n" +
		"  stop                     - Execute ~/bin/stop on content server under this account\n" +
		"  version                  - Display the version of webtools CLI in use\n" +
		"\n" +
		"Environment variables that affect webtools operation, default is [value]:\n" +
		"WT_DEBUG            - Set to 1 to enable debugging output [0]\n" +
		"WT_DEBUGLVL         - Values between 0 (default) and 4 (very verbose)\n" +
		"WT_MODE             - Comma delimited list of services to offer, values include:\n" +
		"                      scheduler, agent\n" +
		"WT_SCHED            - Connection string to Webtools scheduler [tcp://localhost:9912]\n" +
		"WT_APPID            - Application identifier [current username]\n" +
		"WT_SCHED_DBPATH     - Path to scheduler DB json file\n" +
		"                      [/usr/local/etc/webtools/scheduler.json\n" +
		"WT_SCHED_LISTENER   - Listen string for ZMQ [tcp://*:9912]\n" +
		"\n")
}

func DoPingAgent(host string) {
	if DEBUG {
		log.Println("DoPingAgent(", host, ")")
	}
}

func DoPingSched() {
	if DEBUG {
		log.Println("DoPingSched()")
	}
	ok, err := SchedulerPing()
	if ok {
		fmt.Println("Scheduler is alive.")
	} else {
		fmt.Println("Scheduler is not responding. [", err.Error(), "]")
	}
}

func DoKill(pid int) {
	if DEBUG {
		log.Println("DoKill(", pid, ")")
	}
}

func DoPs() {
	if DEBUG {
		log.Println("DoPs()")
	}

}

func DoStart() {
	if DEBUG {
		log.Println("DoStart()")
	}

}

func DoStop() {
	if DEBUG {
		log.Println("DoStop()")
	}

}

func DoSchedLookup(appid string) {
	if DEBUG {
		log.Println("DoSchedLookup()")
	}

	addr, err := SchedulerReqLookup(appid)
	if err != nil {
		fmt.Printf("Scheduler lookup failed for AppID = %s: %s\n", appid, err.Error())
	} else {
		fmt.Printf("The agent for AppID=%s is at %s\n", appid, addr)
	}

}

func DoVersion() {

	fmt.Println("Webtools Version: ", Version)
	maj, min, patch := zmq.Version()
	fmt.Printf("0MQ Version: %d.%d.%d\n", maj, min, patch)

}
