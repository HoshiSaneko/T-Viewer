package adb

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type ProcessInfo struct {
	PID     string `json:"pid"`
	Package string `json:"package"`
}

// ... (ParseDevices and other functions remain the same) ...

type Device struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Model  string `json:"model"`
}

type UINode struct {
	ID            string    `xml:"-" json:"id"`
	Index         string    `xml:"index,attr" json:"index"`
	Text          string    `xml:"text,attr" json:"text"`
	Class         string    `xml:"class,attr" json:"class"`
	Package       string    `xml:"package,attr" json:"package"`
	ContentDesc   string    `xml:"content-desc,attr" json:"contentDesc"`
	Checkable     string    `xml:"checkable,attr" json:"checkable"`
	Checked       string    `xml:"checked,attr" json:"checked"`
	Clickable     string    `xml:"clickable,attr" json:"clickable"`
	Enabled       string    `xml:"enabled,attr" json:"enabled"`
	Focusable     string    `xml:"focusable,attr" json:"focusable"`
	Focused       string    `xml:"focused,attr" json:"focused"`
	Scrollable    string    `xml:"scrollable,attr" json:"scrollable"`
	LongClickable string    `xml:"long-clickable,attr" json:"longClickable"`
	Password      string    `xml:"password,attr" json:"password"`
	Selected      string    `xml:"selected,attr" json:"selected"`
	Bounds        string    `xml:"bounds,attr" json:"bounds"`
	Children      []*UINode `xml:"node" json:"children"`
}

type UIHierarchy struct {
	XMLName  xml.Name `xml:"hierarchy"`
	Rotation string   `xml:"rotation,attr"`
	Root     *UINode  `xml:"node"`
}

// ParseDevices parses the output of adb devices -l
func ParseDevices(output string) []Device {
	lines := strings.Split(output, "\n")
	var devices []Device
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			id := parts[0]
			status := parts[1]
			model := ""
			for _, part := range parts[2:] {
				if strings.HasPrefix(part, "model:") {
					model = strings.TrimPrefix(part, "model:")
				}
			}
			devices = append(devices, Device{
				ID:     id,
				Status: status,
				Model:  model,
			})
		}
	}
	return devices
}

func GetDevices() ([]Device, error) {
	cmd := newCommand("adb", "devices", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute adb devices: %w", err)
	}
	return ParseDevices(out.String()), nil
}

func TakeScreenshot(deviceID string) (string, error) {
	cmd := newCommand("adb", "-s", deviceID, "exec-out", "screencap", "-p")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to take screenshot: %w", err)
	}

	// Convert to base64
	base64Str := base64.StdEncoding.EncodeToString(out.Bytes())
	return "data:image/png;base64," + base64Str, nil
}

// ParseUIHierarchy parses the XML dump
func ParseUIHierarchy(reader io.Reader) (*UINode, error) {
	var hierarchy UIHierarchy
	err := xml.NewDecoder(reader).Decode(&hierarchy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UI dump XML: %w", err)
	}
	assignIDs(hierarchy.Root, "0")
	return hierarchy.Root, nil
}

func GetUIHierarchy(deviceID string) (*UINode, error) {
	// 1. Dump UI hierarchy to a file on device
	dumpCmd := newCommand("adb", "-s", deviceID, "shell", "uiautomator", "dump", "/data/local/tmp/window_dump.xml")

	// Capture stderr for better error reporting, especially for uiautomator conflicts
	var dumpErr bytes.Buffer
	dumpCmd.Stderr = &dumpErr

	err := dumpCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to dump UI hierarchy: %w. stderr: %s", err, dumpErr.String())
	}

	// 2. Read the file
	catCmd := newCommand("adb", "-s", deviceID, "shell", "cat", "/data/local/tmp/window_dump.xml")
	var out bytes.Buffer
	catCmd.Stdout = &out
	err = catCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to read UI dump: %w", err)
	}

	return ParseUIHierarchy(&out)
}

// CheckUiAutomatorProcess checks if uiautomator is currently running and returns the process info
func CheckUiAutomatorProcess(deviceID string) (*ProcessInfo, error) {
	// Method 1: Check dumpsys for instrumentation processes (often used by appium/uiautomator2)
	cmdDump := newCommand("adb", "-s", deviceID, "shell", "dumpsys activity processes | grep -E 'Proc.*instrumentation'")
	var outDump bytes.Buffer
	cmdDump.Stdout = &outDump
	_ = cmdDump.Run()

	dumpOutput := outDump.String()
	dumpLines := strings.Split(dumpOutput, "\n")

	for _, line := range dumpLines {
		line = strings.TrimSpace(line)
		// Example format:
		// Proc # 7: fg     F/ /FGS  -CMNFU-T  t: 0 2132:com.example.testgroup/u0a246 (instrumentation)
		if strings.Contains(line, "instrumentation") && strings.Contains(line, "Proc #") {
			// Extract PID and Package
			// Usually after "t: 0 " or similar, format is PID:PACKAGE/USER
			parts := strings.Split(line, " ")
			for _, part := range parts {
				if strings.Contains(part, ":") && strings.Contains(part, "/") {
					// Likely "2132:com.example.testgroup/u0a246"
					subParts := strings.Split(part, ":")
					if len(subParts) >= 2 {
						pid := subParts[0]
						pkgAndUser := subParts[1]
						pkg := strings.Split(pkgAndUser, "/")[0]

						// Check if pid is numeric
						isNum := true
						for _, char := range pid {
							if char < '0' || char > '9' {
								isNum = false
								break
							}
						}

						if isNum && pid != "" && pkg != "" {
							return &ProcessInfo{
								PID:     pid,
								Package: pkg,
							}, nil
						}
					}
				}
			}
		}
	}

	// Method 2: Check ps for direct uiautomator or app_process
	// First try with -A (modern android)
	cmd := newCommand("adb", "-s", deviceID, "shell", "ps -A")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// If it fails, fallback to simple ps (older android)
	if err != nil || len(out.String()) < 10 {
		cmd = newCommand("adb", "-s", deviceID, "shell", "ps")
		out.Reset()
		cmd.Stdout = &out
		_ = cmd.Run()
	}

	output := out.String()
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "* daemon") || strings.Contains(line, "grep") {
			continue
		}

		// Only look for uiautomator related processes
		if !strings.Contains(strings.ToLower(line), "uiautomator") && !strings.Contains(strings.ToLower(line), "app_process") {
			continue
		}

		// Expected format something like:
		// USER      PID   PPID  VSIZE  RSS   WCHAN            PC  NAME
		// u0_a240   26356 582   16789080 131452 0           0 S com.github.uiautomator
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pid := ""
			pkg := ""

			// Try to intelligently find PID (usually the first number)
			for _, part := range parts {
				isNum := true
				for _, char := range part {
					if char < '0' || char > '9' {
						isNum = false
						break
					}
				}
				if isNum && pid == "" {
					pid = part
				}
			}

			// Package name is usually the last column
			pkg = parts[len(parts)-1]

			// If it's app_process, we need to make sure it's actually running uiautomator
			// Sometimes app_process is used to spawn uiautomator via commands
			if strings.Contains(pkg, "app_process") && !strings.Contains(strings.ToLower(line), "uiautomator") {
				continue
			}

			if pid != "" && !strings.Contains(pkg, "grep") && !strings.Contains(pkg, "ps") {
				return &ProcessInfo{
					PID:     pid,
					Package: pkg,
				}, nil
			}
		}
	}

	return nil, nil // Not found
}

// KillProcess kills a specific process by PID on the device, or by Package Name
func KillProcess(deviceID string, pid string, pkg string) error {
	var cmd *exec.Cmd

	// Since kill -9 often lacks permission, we completely rely on am force-stop
	// for any identified package, and ignore numeric PID kills.

	if pkg != "" && !strings.Contains(pkg, "app_process") && pkg != "uiautomator" {
		cmd = newCommand("adb", "-s", deviceID, "shell", "am", "force-stop", pkg)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to force-stop package %s: %w", pkg, err)
		}
		return nil
	}

	// If we only know it's uiautomator or app_process, try to stop the generic test runners
	// that might be causing it, or fallback to returning an error to prompt reboot.
	cmd = newCommand("adb", "-s", deviceID, "shell", "am", "force-stop", "com.github.uiautomator")
	_ = cmd.Run()
	cmd = newCommand("adb", "-s", deviceID, "shell", "am", "force-stop", "com.github.uiautomator.test")
	_ = cmd.Run()
	cmd = newCommand("adb", "-s", deviceID, "shell", "am", "force-stop", "io.appium.uiautomator2.server")
	_ = cmd.Run()
	cmd = newCommand("adb", "-s", deviceID, "shell", "am", "force-stop", "io.appium.uiautomator2.server.test")
	_ = cmd.Run()

	// If pkg is empty or generic, and we just tried standard ones, we consider it "done"
	// but it might not actually be stopped.
	return nil
}

// RebootDevice reboots the device
func RebootDevice(deviceID string) error {
	cmd := newCommand("adb", "-s", deviceID, "reboot")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to reboot device: %w", err)
	}
	return nil
}

func assignIDs(node *UINode, prefix string) {
	if node == nil {
		return
	}
	node.ID = prefix
	for i, child := range node.Children {
		assignIDs(child, fmt.Sprintf("%s-%d", prefix, i))
	}
}
