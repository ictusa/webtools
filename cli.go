//webtools cli parser
package main

import (
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"log"
	"strconv"
)

const (
	CliInit = iota
	CliHelp
	CliKill
	CliPing
	CliPingSched
	CliPingAgent
	CliPs
	CliScheduler
	CliSchedulerLookup
	CliStart
	CliStop
	CliVersion
	CliService
)

// ParseCli implements a very naive parser for command line arguments.
func ParseCli(cmds []string) {
	if config.Debug {
		log.Println("ParseCli(", cmds, ")")
	}
	parsecli(CliInit, cmds)

}

// parsecli is the recursive portion of the parser, should be called by ParseCli only.
func parsecli(state int, cmds []string) {
	var statemap = map[int]string{
		CliInit:            "CliInit",
		CliHelp:            "CliHelp",
		CliKill:            "CliKill",
		CliPing:            "CliPing",
		CliPingSched:       "CliPingSched",
		CliPingAgent:       "CliPingAgent",
		CliPs:              "CliPs",
		CliScheduler:       "CliScheduler",
		CliSchedulerLookup: "CliSchedulerLookup",
		CliService:         "CliService",
		CliStart:           "CliStart",
		CliStop:            "CliStop",
		CliVersion:         "CliVersion",
	}
	if config.Debug {
		log.Println("parsecli(", statemap[state], ",", cmds, ")")
	}
	switch {
	case state == CliInit:

		switch {
		case cmds[0] == "version":
			parsecli(CliVersion, cmds[1:len(cmds)])

		case cmds[0] == "ping":
			parsecli(CliPing, cmds[1:len(cmds)])

		case cmds[0] == "kill":
			parsecli(CliKill, cmds[1:len(cmds)])

		case cmds[0] == "help":
			parsecli(CliHelp, cmds[1:len(cmds)])

		case cmds[0] == "scheduler":
			parsecli(CliScheduler, cmds[1:len(cmds)])

		case cmds[0] == "service":
			parsecli(CliService, cmds[1:len(cmds)])

		case cmds[0] == "start":
			parsecli(CliStart, cmds[1:len(cmds)])

		case cmds[0] == "stop":
			parsecli(CliStop, cmds[1:len(cmds)])

		case cmds[0] == "ps":
			parsecli(CliPs, cmds[1:len(cmds)])

		default:
			fmt.Println("webtools: unknown command:", cmds[0])
			fmt.Println("Run 'webtools help' for usage information.")
		} // state == CliInit

	case state == CliVersion:
		DoVersion()

	case state == CliPing:
		switch {
		case cmds[0] == "agent":
			parsecli(CliPingAgent, cmds[1:len(cmds)])
		case cmds[0] == "scheduler":
			parsecli(CliPingSched, cmds[1:len(cmds)])

		} // state == CliPing

	case state == CliPingAgent:
		if len(cmds) != 1 {
			log.Fatalln("Usage: webtools ping agent <hostname>")
		}
		DoPingAgent(cmds[0])
	case state == CliPingSched:
		DoPingSched()

	case state == CliKill:
		if len(cmds) != 1 {
			log.Fatalln("Usage: webtools kill <pid>")
		}

		if pid, err := strconv.Atoi(cmds[0]); err == nil {
			DoKill(pid)
		} else {
			log.Fatalln("Usage: webtools kill <pid>, ", err.Error())
		}

	case state == CliScheduler:
		if len(cmds) < 1 {
			DoHelp()
		}
		switch {
		case cmds[0] == "lookup":
			if len(cmds) == 2 {
				DoSchedLookup(cmds[1])
			} else {
				DoSchedLookup(config.AppId)
			}
		default:
			DoHelp()
		}
	case state == CliService:
		if len(cmds) < 1 {
			DoHelp()
			return
		}
		switch {
		case cmds[0] == "agent":
			DoStartAgent()
		case cmds[0] == "scheduler":
			DoStartScheduler()
		default:
			DoHelp()
			return
		}
		if len(cmds) > 1 {
			parsecli(CliService, cmds[1:len(cmds)])
		}
	case state == CliHelp:
		DoHelp()

	case state == CliPs:
		DoPs()

	case state == CliStart:
		DoStart()

	case state == CliStop:
		DoStop()

	} //state switch
}
func DoStartAgent() {
	ServicesRunning = true
	go AgentService()
}
func DoStartScheduler() {
	go SchedulerSigHUPHandler()
	go SchedulerService()
	ServicesRunning = true

}
func DoHelp() {
	fmt.Print("Webtools is an automation tool for use by developers to run commands remotely on content servers.\n" +
		"\n" +
		"Usage:\n" +
		"webtools command <required arguments> [optional arguments] \n" +
		"\n" +
		"The commands are:\n" +
		"  help                      - Display this text\n" +
		"  kill <pid>                - Kill PID on content server\n" +
		"  ping scheduler            - Display status of scheduler\n" +
		"  ping agent <agent addr>   - Display status of agent at connect string\n" +
		"  ps                        - Display processes on content server \n" +
		"  scheduler lookup [Appid]  - Query scheduler for agent address of App\n" +
		"  service <agent|scheduler> - Start agent or scheduler or both\n" +
		"  start                     - Execute ~/bin/start on content server \n" +
		"  stop                      - Execute ~/bin/stop on content server \n" +
		"  version                   - Display the version of webtools CLI in use\n" +
		"\n" +
		"Environment variables that affect webtools operation, default is [value]:\n" +
		"WT_DEBUG            - Set to true to enable debugging output [false]\n" +
		"WT_SCHEDULERADDRESS - Connection string to Webtools scheduler [tcp://localhost:9912]\n" +
		"WT_APPID            - Application identifier [current username]\n" +
		"WT_SCHEDULERDBPATH  - Path to scheduler DB json file\n" +
		"                      [/usr/local/etc/webtools/scheduler.json\n" +
		"WT_SCHEDULERLISTEN  - Listen string for 0MQ [tcp://*:9912]\n" +
		"WT_AGENTLISTEN      - Listen string for 0MQ [tcp://*:9924]\n" +
		"WT_AGENTTIMEOUT     - Wait how long for agent response [30]\n" +
		"\n")
}

func DoPingAgent(host string) {
	if config.Debug {
		log.Println("DoPingAgent(", host, ")")
	}
	ok, err := AgentPing(host)
	if ok {
		fmt.Println("Agent is alive.")
	} else {
		fmt.Println("Agent is not responding. [", err.Error(), "]")
	}
}

func DoPingSched() {
	if config.Debug {
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
	if config.Debug {
		log.Println("DoKill(", pid, ")")
	}
	output, err := AgentReqKill(config.AppId, pid)
	if err != nil {
		fmt.Println("kill failed.")
		fmt.Println(output)
		fmt.Println(err)
		return
	}
	fmt.Println(output)
}

func DoPs() {
	if config.Debug {
		log.Println("DoPs()")
	}
	output, err := AgentReqPs(config.AppId)
	if err != nil {
		fmt.Println("ps failed.")
		fmt.Println(output)
		fmt.Println(err)
		return
	}
	fmt.Println(output)

}

func DoStart() {
	if config.Debug {
		log.Println("DoStart()")
	}
	output, err := AgentReqStartApp(config.AppId)
	if err != nil {
		fmt.Println("App start failed.")
		fmt.Println(output)
		fmt.Println(err)
		return
	}
	fmt.Println("App start ok.")
	fmt.Println(output)

}

func DoStop() {
	if config.Debug {
		log.Println("DoStop()")
	}
	output, err := AgentReqStopApp(config.AppId)
	if err != nil {
		fmt.Println("App stop failed.")
		fmt.Println(output)
		fmt.Println(err)
		return
	}
	fmt.Println("App stop ok.")
	fmt.Println(output)

}

func DoSchedLookup(appid string) {
	if config.Debug {
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
