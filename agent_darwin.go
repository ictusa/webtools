package main

import (
	"os/user"
)

func AgentPs(appid string) (string, error) {
	u, err := user.Lookup(appid)
	if err != nil {
		return "", err
	}

	return runCommand(u, []string{"/bin/ps", "-U", u.Username, "-f", "-x"}, "")
}

func AgentKillPid(appid string, pid string) (string, error) {
	u, err := user.Lookup(appid)
	if err != nil {
		return "", err
	}

	return runCommand(u, []string{"/bin/kill", pid}, "")

}
