package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"haproxy-monitor/watcher"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
)

var pidFile = flag.String("pidFile", "", "path to monitored process's pid file")

func main() {
	cflager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, _ := cflager.New("haproxy-monitor")
	if *pidFile == "" {
		logger.Error("flag-parsing", errors.New("pidfile-not-found"))
		os.Exit(1)
	}

	logger.Info("starting-monitor", lager.Data{"pid-file": *pidFile})

	for {

		pid, err := getPid(*pidFile)
		if err != nil {
			logger.Error("exiting", err)
			os.Exit(1)
		}
		logger.Debug("checking-pid", lager.Data{"pid": pid})
		if !watcher.Running(pid) {
			logger.Error("exiting", fmt.Errorf("PID %d not found", pid))
			os.Exit(1)
		}
		time.Sleep(time.Second)
	}
}

func getPid(pidFile string) (int, error) {
	f, err := os.Open(pidFile)
	if err != nil {
		return -1, fmt.Errorf("Cannot open file %s: %s", pidFile, err.Error())
	}
	defer f.Close()

	retries := 3
	for i := 0; i < retries; i++ {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return -1, fmt.Errorf("Cannot acquire lock: %s", err.Error())
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return -1, fmt.Errorf("Cannot read file %s: %s", pidFile, err.Error())
	}
	data := strings.TrimSpace(string(fileBytes))
	pid, err := strconv.Atoi(data)
	if err != nil {
		return -1, fmt.Errorf("Cannot convert file %s to integer: Contents: %s", pidFile, data)
	}

	return pid, nil
}
