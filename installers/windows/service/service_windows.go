//go:build windows
// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "AgentFS"
const serviceDesc = "AgentFS Semantic Filesystem Service"

var elog debug.Log

type agentFSService struct{}

func (m *agentFSService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Start AgentFS daemon
	go runAgentFS()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				// Graceful shutdown would go here
				return
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
}

func runAgentFS() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		elog.Error(1, fmt.Sprintf("failed to get executable path: %v", err))
		return
	}

	exeDir := filepath.Dir(exePath)
	agentfsPath := filepath.Join(exeDir, "agentfs.exe")

	// Run main agentfs executable
	// In production, this would exec the actual daemon
	elog.Info(1, fmt.Sprintf("Starting AgentFS from %s", agentfsPath))

	// Placeholder - actual implementation would run the daemon
	for {
		time.Sleep(time.Hour)
	}
}

func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &agentFSService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	s, err = m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName: serviceName,
		Description: serviceDesc,
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return err
	}
	defer s.Close()

	// Set up event logging
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource failed: %s", err)
	}

	return nil
}

func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(serviceName)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource failed: %s", err)
	}

	return nil
}

func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("could not send stop: %v", err)
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(timeout) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}

	return nil
}

func main() {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	if !isIntSess {
		runService(serviceName, false)
		return
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: agentfs-service <command>")
		fmt.Println("Commands: install, remove, start, stop, debug")
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "debug":
		runService(serviceName, true)
	case "install":
		err = installService()
	case "remove":
		err = removeService()
	case "start":
		err = startService()
	case "stop":
		err = stopService()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("failed to %s service: %v", cmd, err)
	}

	fmt.Printf("Service %s completed successfully\n", cmd)
}
