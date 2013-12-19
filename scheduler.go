//
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	zmq "github.com/pebbe/zmq4"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var schedulerDbMutex sync.Mutex

//SchedulerDB maps AppID to Agent connect string
var SchedulerDB map[string]string

//SchedLookup, SchedReply, SchedSet, SchedOk, SchedError, SchedUnknown, SchedNotFound, SchedPing
//SchedPingReply are constants used in request specific actions from the scheduler by the CLI
//in 0MQ messages.
const (
	SchedLookup = iota
	SchedReply
	SchedSet
	SchedOk
	SchedError
	SchedUnknown
	SchedNotFound
	SchedPing
	SchedPingReply
)

//SchedulerMsg is a struct that represents requests and responses between the scheduler and CLI.
//They are sent JSON encoded as 0MQ messages.
type SchedulerMsg struct {
	MsgType int
	AppID   string
	Address string
	Error   string
}

func init() {
	SchedulerDB = make(map[string]string)
}

//LoadSchedulerDB will load the SchedulerDB map from the specified JSON file.
func LoadSchedulerDB(path string) error {
	if config.Debug {
		log.Println("LoadSchedulerDB(", path, ")")
	}
	db, openErr := os.Open(path)
	if openErr != nil {
		return openErr
	}
	defer db.Close()

	info, statErr := db.Stat()
	if statErr != nil {
		return statErr
	}

	var in = make([]byte, info.Size())

	_, readErr := db.Read(in)
	if readErr != nil {
		return readErr
	}
	if config.Debug == true && config.DebugLvl > 3 {
		log.Print(in)
	}

	// NB: maps in Go are not safe to manipulate concurrently.
	schedulerDbMutex.Lock()
	defer schedulerDbMutex.Unlock()
	marshalErr := json.Unmarshal(in, &SchedulerDB)
	return marshalErr
}

//SchedulerLookup is a wrapper around the SchedulerDB map to synchronize access
func SchedulerLookup(appid string) (string, bool) {
	schedulerDbMutex.Lock()
	defer schedulerDbMutex.Unlock()
	agent, ok := SchedulerDB[appid]
	return agent, ok
}

//SchedulerSigHUPHandler causes the SchedulerDB to be reloaded on receipt of SIGHUP. Should be run as a separate go routine.
func SchedulerSigHUPHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	for {
		<-c //block until we receive SIGHUP
		log.Println("Reloading SchedulerDB SIGHUP received.")
		LoadSchedulerDB(config.SchedulerDbPath)
	}
}

//SchedulerService provides all the network based services of the Scheduler. It creates
//the 0MQ listener, and responds to queries. It is intended to be run inside a go routine, it
//does not return to its caller.
func SchedulerService() {
	if config.Debug {
		log.Println("SchedulerService()")
	}

	if err := LoadSchedulerDB(config.SchedulerDbPath); err != nil {
		log.Fatalln("LoadSchedulerDB: ", err)
	}

	responder, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		log.Fatalln("SchedulerService() 0MQ NewSocket:", err.Error())
	}
	defer responder.Close()

	if err := responder.Bind(config.SchedulerListen); err != nil {
		log.Fatalln("SchedulerService():responder.Bind(", config.SchedulerListen, ")", err.Error())
	}

	for {
		msg, err := responder.RecvBytes(0) //Flags for Recv?
		if config.Debug {
			log.Println("SchedulerService() 0MQ Recv:", bytes.NewBuffer(msg).String())
		}

		if config.Debug && err != nil {
			log.Println("SchedulerService() 0MQ Recv error: ", err.Error())
		}

		if err != nil {
			continue
		}
		var Query SchedulerMsg
		var Reply SchedulerMsg
		if err := json.Unmarshal(msg, &Query); err != nil {
			Reply = SchedulerMsg{SchedError, "", "", err.Error()}
			b, _ := json.Marshal(Reply)
			responder.SendBytes(b, 0)
			continue
		}
		switch {
		case Query.MsgType == SchedLookup:
			agent, ok := SchedulerLookup(Query.AppID)
			if ok == true {
				Reply = SchedulerMsg{SchedReply, Query.AppID, agent, ""}
			} else {
				Reply = SchedulerMsg{SchedNotFound, Query.AppID, "", ""}
			}
		case Query.MsgType == SchedPing:
			Reply = SchedulerMsg{SchedPingReply, "", "", ""}
		default:
			Reply = SchedulerMsg{SchedUnknown, "", "", ""}
		}

		b, _ := json.Marshal(Reply)
		responder.SendBytes(b, 0)

	} //end for{}
}

//SchedulerReqLookup sends a LOOKUP request to the scheduler defined in WT_SCHED env variable.
//Returns the agent string on success, and "" with an error on failure. Uses a 1 second timeout.
func SchedulerReqLookup(appid string) (string, error) {
	if config.Debug {
		log.Printf("SchedulerReqLookup(%s) to %s\n", appid, config.SchedulerAddress)
	}
	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return "", err
	}
	if config.Debug {
		log.Println("SchedulerReqLookup() 0MQ NewSocket(zmq.REQ) ok")
	}
	defer requester.Close()

	connErr := requester.Connect(config.SchedulerAddress)
	if connErr != nil {
		return "", connErr
	}

	poller := zmq.NewPoller()
	poller.Add(requester, zmq.POLLIN)
	var msg SchedulerMsg
	msg = SchedulerMsg{SchedLookup, appid, "", ""}
	jsonOut, jsonErr := json.Marshal(msg)
	if jsonErr != nil {
		return "", jsonErr
	}

	byteSent, sendErr := requester.SendBytes(jsonOut, 0)
	if sendErr != nil {
		log.Println("SchedulerReqLookup() 0MQ SendBytes:", sendErr)
		return "", sendErr
	}
	if config.Debug {
		log.Println("SchedulerReqLookup() 0MQ SendBytes sent ", byteSent)
	}
	//Poll socket for a reply, with a timeout
	sockets, pollerErr := poller.Poll(1000 * time.Millisecond)
	if pollerErr != nil {
		return "", pollerErr // Interrupted by a syscall?
	}

	// Process the server reply. If we didn't get a reply close the socket and fail.
	if len(sockets) > 0 { //We got something
		reply, zmqErr := requester.RecvBytes(0)
		if zmqErr != nil {
			return "", zmqErr
		}
		if config.Debug {
			log.Println("SchedulerReqLookup() 0MQ Recv msg:", bytes.NewBuffer(reply).String())
		}

		jsonErr = json.Unmarshal(reply, &msg)
		if jsonErr != nil {
			return "", jsonErr
		}

		switch {

		case msg.MsgType == SchedReply:
			return msg.Address, nil
		case msg.MsgType == SchedNotFound:
			return "", errors.New("AppID not found")
		default:
			return "", errors.New(msg.Error)
		}

	} else {
		return "", errors.New("timeout")
	}

}

//SchedulerPing sends a PING request to the scheduler defined in WT_SCHED env variable. Returns true
//on success, and false with an error on failure. Uses a 1 second timeout.
func SchedulerPing() (bool, error) {
	if config.Debug {
		log.Println("SchedulerPing() to ", config.SchedulerAddress)
	}

	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return false, err
	}
	if config.Debug {
		log.Println("SchedulerPing() 0MQ NewSocket(zmq.REQ) ok")
	}
	defer requester.Close()

	connErr := requester.Connect(config.SchedulerAddress)
	if connErr != nil {
		return false, connErr
	}

	poller := zmq.NewPoller()
	poller.Add(requester, zmq.POLLIN)

	var msg SchedulerMsg
	msg = SchedulerMsg{SchedPing, "", "", ""}
	jsonOut, jsonErr := json.Marshal(msg)
	if jsonErr != nil {
		return false, jsonErr
	}

	byteSent, sendErr := requester.SendBytes(jsonOut, 0)
	if sendErr != nil {
		log.Fatalln("SchedulerPing() 0MQ SendBytes:", sendErr.Error())
	}
	if config.Debug {
		log.Println("SchedulerPing() 0MQ SendBytes sent ", byteSent)
	}

	//Poll socket for a reply, with a timeout
	sockets, pollerErr := poller.Poll(1000 * time.Millisecond)
	if pollerErr != nil {
		return false, pollerErr // Interrupted by a syscall?
	}

	// Process the server reply. If we didn't get a reply close the socket and fail.
	if len(sockets) > 0 {
		reply, zmqErr := requester.RecvBytes(0)
		if zmqErr != nil {
			return false, zmqErr
		}
		if config.Debug {
			log.Println("SchedulerPing() 0MQ Recv msg:", bytes.NewBuffer(reply).String())
		}

		jsonErr = json.Unmarshal(reply, &msg)
		if jsonErr != nil {
			return false, jsonErr
		}

		if msg.MsgType == SchedPingReply {
			return true, nil
		}
		return false, errors.New(msg.Error)

	}
	return false, errors.New("timeout")

}
