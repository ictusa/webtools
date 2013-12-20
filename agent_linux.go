package main

import (
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
