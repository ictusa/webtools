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

var SchedulerDBMutex sync.Mutex
var SchedulerDB map[string]string
var SchedulerDBPath string
var SchedulerListen string

const (
	SCHED_LOOKUP = iota
	SCHED_REPLY
	SCHED_SET
	SCHED_OK
	SCHED_ERROR
	SCHED_UNKNOWN
	SCHED_NOTFOUND
	SCHED_PING
	SCHED_PINGREPLY
)

type SchedulerMsg struct {
	MsgType int
	AppID   string
	Address string
	Error   string
}

func init() {
	SchedulerDB = make(map[string]string)

	if val := os.Getenv("WT_SCHED_DBPATH"); val != "" {
		SchedulerDBPath = val
	} else {
		SchedulerDBPath = "/usr/local/etc/webtools/scheduler.json"
	}

	if val := os.Getenv("WT_SCHED_LISTENER"); val != "" {
		SchedulerListen = val
	} else {
		SchedulerListen = "tcp://*:9912"
	}

}

//LoadSchedulerDB(path) will load the SchedulerDB map from the specified JSON file.
func LoadSchedulerDB(path string) error {
	if DEBUG {
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

	var in []byte = make([]byte, info.Size())

	_, readErr := db.Read(in)
	if readErr != nil {
		return readErr
	}
	if DEBUG == true && DEBUGLVL > 3 {
		log.Print(in)
	}

	// NB: maps in Go are not safe to manipulate concurrently.
	SchedulerDBMutex.Lock()
	defer SchedulerDBMutex.Unlock()
	marshalErr := json.Unmarshal(in, &SchedulerDB)
	return marshalErr
}

func SchedulerLookup(appid string) (string, bool) {
	SchedulerDBMutex.Lock()
	defer SchedulerDBMutex.Unlock()
	agent, ok := SchedulerDB[appid]
	return agent, ok
}

func SchedulerSigHUPHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	for {
		<-c //block until we receive SIGHUP
		log.Println("Reloading SchedulerDB SIGHUP received.")
		LoadSchedulerDB(SchedulerDBPath)
	}
}

//SchedulerService provides all the network based services of the Scheduler. It creates
//the 0MQ listener, and responds to queries. It is intended to be run inside a go routine, it
//does not return to its caller.
func SchedulerService() {
	if DEBUG {
		log.Println("SchedulerService()")
	}

	if err := LoadSchedulerDB(SchedulerDBPath); err != nil {
		log.Fatalln("LoadSchedulerDB: ", err)
	}

	responder, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		log.Fatalln("SchedulerService() 0MQ NewSocket:", err.Error())
	}
	defer responder.Close()

	if err := responder.Bind(SchedulerListen); err != nil {
		log.Fatalln("SchedulerService():responder.Bind(", SchedulerListen, ")", err.Error())
	}

	for {
		msg, err := responder.RecvBytes(0) //Flags for Recv?
		if DEBUG {
			log.Println("SchedulerService() 0MQ Recv:", bytes.NewBuffer(msg).String())
		}

		if DEBUG && err != nil {
			log.Println("SchedulerService() 0MQ Recv error: ", err.Error())
		}

		if err != nil {
			continue
		}
		var Query SchedulerMsg
		var Reply SchedulerMsg
		if err := json.Unmarshal(msg, &Query); err != nil {
			Reply = SchedulerMsg{SCHED_ERROR, "", "", err.Error()}
			b, _ := json.Marshal(Reply)
			responder.SendBytes(b, 0)
			continue
		}
		switch {
		case Query.MsgType == SCHED_LOOKUP:
			agent, ok := SchedulerLookup(Query.AppID)
			if ok == true {
				Reply = SchedulerMsg{SCHED_REPLY, Query.AppID, agent, ""}
			} else {
				Reply = SchedulerMsg{SCHED_NOTFOUND, Query.AppID, "", ""}
			}
		case Query.MsgType == SCHED_PING:
			Reply = SchedulerMsg{SCHED_PINGREPLY, "", "", ""}
		default:
			Reply = SchedulerMsg{SCHED_UNKNOWN, "", "", ""}
		}

		b, _ := json.Marshal(Reply)
		responder.SendBytes(b, 0)

	} //end for{}
}

//SchulderReqLookup(appid) sends a LOOKUP request to the scheduler defined in WT_SCHED env variable.
//Returns the agent string on success, and "" with an error on failure. Uses a 1 second timeout.
func SchedulerReqLookup(appid string) (string, error) {
	if DEBUG {
		log.Printf("SchedulerReqLookup(%s) to %s\n", appid, SchedulerAddress)
	}
	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return "", err
	}
	if DEBUG {
		log.Println("SchedulerReqLookup() 0MQ NewSocket(zmq.REQ) ok")
	}
	defer requester.Close()

	connErr := requester.Connect(SchedulerAddress)
	if connErr != nil {
		return "", connErr
	}

	poller := zmq.NewPoller()
	poller.Add(requester, zmq.POLLIN)
	var msg SchedulerMsg
	msg = SchedulerMsg{SCHED_LOOKUP, appid, "", ""}
	jsonOut, jsonErr := json.Marshal(msg)
	if jsonErr != nil {
		return "", jsonErr
	}

	byteSent, sendErr := requester.SendBytes(jsonOut, 0)
	if sendErr != nil {
		log.Println("SchedulerReqLookup() 0MQ SendBytes:", sendErr)
		return "", sendErr
	}
	if DEBUG {
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
		if DEBUG {
			log.Println("SchedulerReqLookup() 0MQ Recv msg:", bytes.NewBuffer(reply).String())
		}

		jsonErr = json.Unmarshal(reply, &msg)
		if jsonErr != nil {
			return "", jsonErr
		}

		switch {

		case msg.MsgType == SCHED_REPLY:
			return msg.Address, nil
		case msg.MsgType == SCHED_NOTFOUND:
			return "", errors.New("AppID not found.")
		default:
			return "", errors.New(msg.Error)
		}

	} else {
		return "", errors.New("Timeout")
	}

}

//SchedulerPing sends a PING request to the scheduler defined in WT_SCHED env variable. Returns true
//on success, and false with an error on failure. Uses a 1 second timeout.
func SchedulerPing() (bool, error) {
	if DEBUG {
		log.Println("SchedulerPing() to ", SchedulerAddress)
	}

	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return false, err
	}
	if DEBUG {
		log.Println("SchedulerPing() 0MQ NewSocket(zmq.REQ) ok")
	}
	defer requester.Close()

	connErr := requester.Connect(SchedulerAddress)
	if connErr != nil {
		return false, connErr
	}

	poller := zmq.NewPoller()
	poller.Add(requester, zmq.POLLIN)

	var msg SchedulerMsg
	msg = SchedulerMsg{SCHED_PING, "", "", ""}
	jsonOut, jsonErr := json.Marshal(msg)
	if jsonErr != nil {
		return false, jsonErr
	}

	byteSent, sendErr := requester.SendBytes(jsonOut, 0)
	if sendErr != nil {
		log.Fatalln("SchedulerPing() 0MQ SendBytes:", sendErr.Error())
	}
	if DEBUG {
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
		if DEBUG {
			log.Println("SchedulerPing() 0MQ Recv msg:", bytes.NewBuffer(reply).String())
		}

		jsonErr = json.Unmarshal(reply, &msg)
		if jsonErr != nil {
			return false, jsonErr
		}

		if msg.MsgType == SCHED_PINGREPLY {
			return true, nil
		} else {
			return false, errors.New(msg.Error)
		}

	} else {
		return false, errors.New("Timeout")
	}

}
