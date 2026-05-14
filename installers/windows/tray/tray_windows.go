//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getlantern/systray"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("StrataFS")
	systray.SetTooltip("StrataFS - Semantic Filesystem")

	// Menu items
	mOpen := systray.AddMenuItem("Open StrataFS", "Open the StrataFS control panel")
	mSearch := systray.AddMenuItem("Search...", "Open search interface")

	systray.AddSeparator()

	mStatus := systray.AddMenuItem("Status: Running", "Current service status")
	mStatus.Disable()

	systray.AddSeparator()

	// Service controls
	mStartService := systray.AddMenuItem("Start Service", "Start the StrataFS service")
	mStopService := systray.AddMenuItem("Stop Service", "Stop the StrataFS service")
	mRestartService := systray.AddMenuItem("Restart Service", "Restart the StrataFS service")

	systray.AddSeparator()

	// Settings submenu
	mSettings := systray.AddMenuItem("Settings", "")
	mOpenConfig := mSettings.AddSubMenuItem("Open Config", "Open configuration file")
	mOpenLogs := mSettings.AddSubMenuItem("View Logs", "Open log directory")
	mAutoStart := mSettings.AddSubMenuItemCheckbox("Start with Windows", "Start StrataFS when Windows starts", true)

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the tray application")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openUI()
			case <-mSearch.ClickedCh:
				openSearch()
			case <-mStartService.ClickedCh:
				runServiceCmd("start")
				mStatus.SetTitle("Status: Running")
			case <-mStopService.ClickedCh:
				runServiceCmd("stop")
				mStatus.SetTitle("Status: Stopped")
			case <-mRestartService.ClickedCh:
				runServiceCmd("stop")
				runServiceCmd("start")
				mStatus.SetTitle("Status: Running")
			case <-mOpenConfig.ClickedCh:
				openConfig()
			case <-mOpenLogs.ClickedCh:
				openLogs()
			case <-mAutoStart.ClickedCh:
				if mAutoStart.Checked() {
					mAutoStart.Uncheck()
					disableAutoStart()
				} else {
					mAutoStart.Check()
					enableAutoStart()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// Cleanup
}

func getIcon() []byte {
	// Return embedded icon bytes
	// In production, this would load from resources
	return []byte{}
}

func openUI() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	uiPath := filepath.Join(exeDir, "stratafs-ui.exe")

	if _, err := os.Stat(uiPath); err == nil {
		exec.Command(uiPath).Start()
	} else {
		// Fall back to web interface
		exec.Command("cmd", "/c", "start", "http://localhost:8080").Start()
	}
}

func openSearch() {
	exec.Command("cmd", "/c", "start", "http://localhost:8080/search").Start()
}

func runServiceCmd(cmd string) {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	servicePath := filepath.Join(exeDir, "stratafs-service.exe")

	exec.Command(servicePath, cmd).Run()
}

func openConfig() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".stratafs", "config.json")
	exec.Command("notepad", configPath).Start()
}

func openLogs() {
	homeDir, _ := os.UserHomeDir()
	logsDir := filepath.Join(homeDir, ".stratafs", "logs")
	os.MkdirAll(logsDir, 0755)
	exec.Command("explorer", logsDir).Start()
}

func enableAutoStart() {
	exePath, _ := os.Executable()

	// Add to registry Run key
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	exec.Command("reg", "add", key, "/v", "StrataFS", "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s"`, exePath), "/f").Run()
}

func disableAutoStart() {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	exec.Command("reg", "delete", key, "/v", "StrataFS", "/f").Run()
}
