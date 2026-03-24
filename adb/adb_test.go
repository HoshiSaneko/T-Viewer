package adb

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func helperCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(0)
	}

	cmd, args := args[0], args[1:]
	if cmd == "adb" {
		if args[0] == "devices" {
			fmt.Print("List of devices attached\ndevice123 device\n")
		} else if args[0] == "-s" && args[2] == "exec-out" {
			fmt.Print("fake_screenshot_data")
		} else if args[0] == "-s" && args[2] == "shell" && args[3] == "uiautomator" {
			// success
		} else if args[0] == "-s" && args[2] == "shell" && args[3] == "cat" {
			fmt.Print(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0"><node index="0" class="test"></node></hierarchy>`)
		}
	}
	os.Exit(0)
}

func TestGetDevices(t *testing.T) {
	execCommand = helperCommand
	defer func() { execCommand = exec.Command }()

	devices, err := GetDevices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != "device123" {
		t.Errorf("unexpected devices: %+v", devices)
	}
}

func TestTakeScreenshot(t *testing.T) {
	execCommand = helperCommand
	defer func() { execCommand = exec.Command }()

	data, err := TakeScreenshot("device123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(data, "data:image/png;base64,") {
		t.Errorf("unexpected screenshot data: %s", data)
	}
}

func TestGetUIHierarchyCmd(t *testing.T) {
	execCommand = helperCommand
	defer func() { execCommand = exec.Command }()

	root, err := GetUIHierarchy("device123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil || root.Class != "test" {
		t.Errorf("unexpected root node")
	}
}

func TestParseDevices(t *testing.T) {
	output := `List of devices attached
device123      device usb:123456X product:Pixel_4 model:Pixel_4 device:flame
device456      offline usb:123456Y
device789      unauthorized
`
	devices := ParseDevices(output)
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}

	if devices[0].ID != "device123" || devices[0].Status != "device" || devices[0].Model != "Pixel_4" {
		t.Errorf("unexpected first device: %+v", devices[0])
	}
	if devices[1].ID != "device456" || devices[1].Status != "offline" {
		t.Errorf("unexpected second device: %+v", devices[1])
	}
	if devices[2].ID != "device789" || devices[2].Status != "unauthorized" {
		t.Errorf("unexpected third device: %+v", devices[2])
	}
}

func TestParseUIHierarchy(t *testing.T) {
	xmlData := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="" class="android.widget.FrameLayout" package="com.android.systemui" content-desc="" checkable="false" checked="false" clickable="false" enabled="true" focusable="false" focused="false" scrollable="false" long-clickable="false" password="false" selected="false" bounds="[0,0][1080,2400]">
    <node index="0" text="Test Node" class="android.widget.TextView" bounds="[100,100][200,200]" />
  </node>
</hierarchy>`

	root, err := ParseUIHierarchy(strings.NewReader(xmlData))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if root == nil {
		t.Fatal("expected root node, got nil")
	}

	if root.Class != "android.widget.FrameLayout" {
		t.Errorf("expected root class android.widget.FrameLayout, got %s", root.Class)
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}

	child := root.Children[0]
	if child.Text != "Test Node" {
		t.Errorf("expected child text 'Test Node', got '%s'", child.Text)
	}
	if child.ID != "0-0" {
		t.Errorf("expected child ID '0-0', got '%s'", child.ID)
	}
}

func TestAssignIDs(t *testing.T) {
	root := &UINode{
		Children: []*UINode{
			{Children: []*UINode{{}}},
			{},
		},
	}
	assignIDs(root, "0")
	if root.ID != "0" {
		t.Errorf("expected root ID 0, got %s", root.ID)
	}
	if root.Children[0].ID != "0-0" {
		t.Errorf("expected child 0 ID 0-0, got %s", root.Children[0].ID)
	}
	if root.Children[0].Children[0].ID != "0-0-0" {
		t.Errorf("expected child 0-0 ID 0-0-0, got %s", root.Children[0].Children[0].ID)
	}
	if root.Children[1].ID != "0-1" {
		t.Errorf("expected child 1 ID 0-1, got %s", root.Children[1].ID)
	}

	// test nil
	assignIDs(nil, "0")
}
