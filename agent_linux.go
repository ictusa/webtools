package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"os/user"
)

func AgentPs(appid string) (string, error) {
	u, err := user.Lookup(appid)
	if err != nil {
		return "", err
	}

	return runCommand(u, []string{"/bin/ps", "-U", u.Username, "u"}, u.HomeDir)
}

func AgentKillPid(appid string, pid string) (string, error) {
	u, err := user.Lookup(appid)
	if err != nil {
		return "", err
	}

	return runCommand(u, []string{"/bin/kill", pid}, "")
}

func runCommand(runas *user.User, cmdLine []string, dir string) (string, error) {
	var err error

	//If they specified a chdir option, do it before executing the command.
	//We do this after the privilege change to ensure the chdir is done with the permissions of the user that the command will
	//run as.
	// currentUid := syscall.Getuid()
	// // currentGid := syscall.Getgid()
	// var wantUid int
	// // var wantGid int
	// wantUid, err = strconv.Atoi(runas.Uid)
	// if err != nil {
	// 	return "", err
	// }

	// if currentUid != wantUid {
	// 	defer restorePriv(currentUid)
	// 	changePriv(wantUid)

	// }
	if dir != "" {
		var pwd string
		pwd, err = os.Getwd()
		if err != nil {
			//log.Println("Getwd()")
			return "", err
		}
		defer os.Chdir(pwd) //Make sure at the end of all this, we go back to our real working directory.
		err = os.Chdir(dir)
		if err != nil {
			log.Println("Chdir(", dir, ")")
			return "", err
		}
	}
	var runUser = []string{"/usr/sbin/runuser", "-", runas.Username}
	runUser = append(runUser, cmdLine...)
	cmd := exec.Command(runUser[0])
	cmd.Args = runUser
	output, cmd_err := cmd.CombinedOutput()
	//Add a timeout mechanism here? Or fancier 0MQ setup that allows concurrent req/rep actions.
	return bytes.NewBuffer(output).String(), cmd_err
}
