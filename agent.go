//
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"log"
	// "os"
	// "os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

const (
	MsgAgentStartApp = iota
	MsgAgentStopApp
	MsgAgentPs
	MsgAgentPing
	MsgAgentPingReply
	MsgAgentKillPid
	MsgAgentForceKillPid
	MsgAgentError
)

//AgentMsg is a struct that represents requests and replies to an agent from the CLI.
//The MsgData field is an operation specific JSON encoded structure.
type AgentMsg struct {
	MsgType int
	AppID   string
	MsgData string
	Error   string
}

func AgentService() {
	if config.Debug {
		log.Println("AgentService()")
	}

	currentUser, userErr := user.Current()
	if userErr != nil {
		log.Fatalln(userErr)
	}
	uid, atoiErr := strconv.Atoi(currentUser.Uid)
	if atoiErr != nil {
		log.Fatalln(atoiErr)
	}

	if uid != 0 {
		log.Fatalln("agent must be run as root")
	}

	responder, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		log.Fatalln("AgentService() 0MQ NewSocket:", err)
	}

	defer responder.Close()
	if err := responder.Bind(config.AgentListen); err != nil {
		log.Fatalln("SchedulerService():responder.Bind(", config.AgentListen, ")", err.Error())
	}
	for {
		msg, err := responder.RecvBytes(0) //Flags for Recv?
		if config.Debug {
			log.Println("AgentService() 0MQ Recv:", bytes.NewBuffer(msg).String())
		}

		if config.Debug && err != nil {
			log.Println("AgentService() 0MQ Recv error: ", err.Error())
		}

		if err != nil {
			continue
		}
		var Query AgentMsg
		var Reply AgentMsg

		if err := json.Unmarshal(msg, &Query); err != nil {
			Reply = AgentMsg{MsgAgentError, "", "", err.Error()}
			b, _ := json.Marshal(Reply)
			responder.SendBytes(b, 0)
			if config.Debug {
				log.Println("AgentService() json Unmarshal: ", err)
			}
			continue
		}

		// var cmdOutput string
		var runErr error
		Reply.MsgType = Query.MsgType

		switch {
		case Query.MsgType == MsgAgentStartApp:
			Reply.MsgData, runErr = AgentStartApp(Query.AppID)

		case Query.MsgType == MsgAgentStopApp:
			Reply.MsgData, runErr = AgentStopApp(Query.AppID)

		case Query.MsgType == MsgAgentPs:
			Reply.MsgData, runErr = AgentPs(Query.AppID)
		case Query.MsgType == MsgAgentKillPid:
			Reply.MsgData, runErr = AgentKillPid(Query.AppID, Query.MsgData)

		case Query.MsgType == MsgAgentPing:
			Reply.MsgType = MsgAgentPingReply

		case Query.MsgType == MsgAgentPingReply:
			runErr = errors.New("malformed agent request")
		}

		if runErr != nil {
			Reply.Error = runErr.Error()
			Reply.MsgType = MsgAgentError
		}
		b, _ := json.Marshal(Reply)
		responder.SendBytes(b, 0)

	} //end for{}

}

func AgentReqStartApp(appid string) (string, error) {
	var req = AgentMsg{MsgAgentStartApp, appid, "", ""}
	agentConnect, err := SchedulerReqLookup(appid)
	if err != nil {
		return "", err
	}

	reply, reqError := AgentReq(&req, agentConnect)
	if reqError != nil {
		return "", errors.New(reply.Error)
	}

	if reply.MsgType != MsgAgentStartApp {
		return reply.MsgData, errors.New(reply.Error)
	}

	return reply.MsgData, nil
}

func AgentReqPs(appid string) (string, error) {
	var req = AgentMsg{MsgAgentPs, appid, "", ""}
	agentConnect, err := SchedulerReqLookup(appid)
	if err != nil {
		return "", err
	}

	reply, reqError := AgentReq(&req, agentConnect)
	if reqError != nil {
		return "", errors.New(reply.Error)
	}

	if reply.MsgType != MsgAgentPs {
		return reply.MsgData, errors.New(reply.Error)
	}

	return reply.MsgData, nil
}

func AgentReqKill(appid string, pid int) (string, error) {
	var req = AgentMsg{MsgAgentKillPid, appid, fmt.Sprintf("%d", pid), ""}
	agentConnect, err := SchedulerReqLookup(appid)
	if err != nil {
		return "", err
	}

	reply, reqError := AgentReq(&req, agentConnect)
	if reqError != nil {
		return "", errors.New(reply.Error)
	}

	if reply.MsgType != MsgAgentKillPid {
		return reply.MsgData, errors.New(reply.Error)
	}

	return reply.MsgData, nil
}

func AgentPing(agentConnect string) (bool, error) {
	var req = AgentMsg{MsgAgentPing, "", "", ""}

	reply, reqError := AgentReq(&req, agentConnect)
	if reqError != nil {
		return false, errors.New(reply.Error)
	}

	if reply.MsgType != MsgAgentPingReply {
		return false, errors.New(reply.Error)
	}

	return true, nil
}

func AgentReqStopApp(appid string) (string, error) {
	var req = AgentMsg{MsgAgentStopApp, appid, "", ""}
	agentConnect, err := SchedulerReqLookup(appid)
	if err != nil {
		return "", err
	}

	reply, reqError := AgentReq(&req, agentConnect)
	if reqError != nil {
		return "", errors.New(reply.Error)
	}

	if reply.MsgType != MsgAgentStopApp {
		return reply.MsgData, errors.New(reply.Error)
	}

	return reply.MsgData, nil
}

func AgentStartApp(appid string) (string, error) {
	u, err := user.Lookup(appid)
	if err != nil {
		return "", err
	}

	return runCommand(u, []string{"bin/start"}, u.HomeDir)
}

func AgentStopApp(appid string) (string, error) {
	u, err := user.Lookup(appid)
	if err != nil {
		return "", err
	}

	return runCommand(u, []string{"bin/stop"}, u.HomeDir)
}

func changePriv(uid int) {
	err := syscall.Setreuid(-1, uid)
	if err != nil {
		log.Println("ChangePriv()", err.Error())
	}

}
func restorePriv(uid int) {
	err := syscall.Setreuid(uid, uid)
	if err != nil {
		log.Println("restorePriv", err.Error())
	}

}

//AgentReq encodes and sends a request to the specified agent, returns the reply.
func AgentReq(req *AgentMsg, agentConnect string) (*AgentMsg, error) {
	if config.Debug {
		log.Printf("AgentReq() to %s\n", agentConnect)
	}

	var reply = AgentMsg{MsgAgentError, "", "", ""}

	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return &reply, err
	}
	if config.Debug {
		log.Println("AgentReq() 0MQ NewSocket(zmq.REQ) ok")
	}
	defer requester.Close()

	connErr := requester.Connect(agentConnect)
	if connErr != nil {
		return &reply, connErr
	}

	poller := zmq.NewPoller()
	poller.Add(requester, zmq.POLLIN)
	jsonOut, jsonErr := json.Marshal(req)
	if jsonErr != nil {
		return &reply, jsonErr
	}

	byteSent, sendErr := requester.SendBytes(jsonOut, 0)
	if sendErr != nil {
		log.Println("AgentReq() 0MQ SendBytes:", sendErr)
		return &reply, sendErr
	}
	if config.Debug {
		log.Println("AgentReq() 0MQ SendBytes sent ", byteSent)
	}
	//Poll socket for a reply, with a timeout
	sockets, pollerErr := poller.Poll(time.Duration(config.AgentTimeout) * time.Second)
	if pollerErr != nil {
		return &reply, pollerErr // Interrupted by a syscall?
	}

	// Process the server reply. If we didn't get a reply close the socket and fail.
	if len(sockets) > 0 { //We got something
		jsonReply, zmqErr := requester.RecvBytes(0)
		if zmqErr != nil {
			return &reply, zmqErr
		}
		if config.Debug {
			log.Println("AgentReq() 0MQ Recv msg:", bytes.NewBuffer(jsonReply).String())
		}

		jsonErr = json.Unmarshal(jsonReply, &reply)
		if jsonErr != nil {
			return &reply, jsonErr
		}
		return &reply, nil

	}
	if config.Debug {
		log.Println("Timeout AgentReq() 0MQ poller.")
	}
	reply.MsgType = MsgAgentError
	return &reply, errors.New("timeout")

}
