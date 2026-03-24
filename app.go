package main

import (
	"T-Viewer/adb"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetDevices returns the list of connected Android devices
func (a *App) GetDevices() ([]adb.Device, error) {
	return adb.GetDevices()
}

// TakeScreenshot takes a screenshot of the specified device and returns a base64 encoded image string
func (a *App) TakeScreenshot(deviceID string) (string, error) {
	return adb.TakeScreenshot(deviceID)
}

// GetUIHierarchy gets the UI hierarchy of the specified device
func (a *App) GetUIHierarchy(deviceID string) (*adb.UINode, error) {
	return adb.GetUIHierarchy(deviceID)
}

// CheckUiAutomatorProcess checks if uiautomator is currently running and returns the process info
func (a *App) CheckUiAutomatorProcess(deviceID string) (*adb.ProcessInfo, error) {
	return adb.CheckUiAutomatorProcess(deviceID)
}

// KillProcess kills a specific process by PID on the device
func (a *App) KillProcess(deviceID string, pid string, pkg string) error {
	return adb.KillProcess(deviceID, pid, pkg)
}

// RebootDevice reboots the specified device
func (a *App) RebootDevice(deviceID string) error {
	return adb.RebootDevice(deviceID)
}

// CropAndSaveScreenshot crops a base64 screenshot and saves it using a native file dialog
func (a *App) CropAndSaveScreenshot(base64Data string, x, y, width, height int, defaultName string) error {
	// Let user select save location
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "PNG Image", Pattern: "*.png"},
		},
	})

	if err != nil {
		return err
	}

	if savePath == "" {
		// User cancelled
		return nil
	}

	// Remove data URI prefix if present
	b64 := base64Data
	if idx := strings.Index(b64, ","); idx != -1 {
		b64 = b64[idx+1:]
	}

	// Decode base64
	imgData, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	// Decode image
	img, _, err := image.Decode(strings.NewReader(string(imgData)))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Define cropping region
	rect := image.Rect(x, y, x+width, y+height)

	// Ensure rect is within image bounds
	bounds := img.Bounds()
	rect = rect.Intersect(bounds)
	if rect.Empty() {
		return fmt.Errorf("crop region is outside image bounds")
	}

	// Create sub-image
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	var croppedImg image.Image
	if simg, ok := img.(subImager); ok {
		croppedImg = simg.SubImage(rect)
	} else {
		return fmt.Errorf("image does not support cropping")
	}

	// Create output file
	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Encode and save as PNG
	if err := png.Encode(out, croppedImg); err != nil {
		return fmt.Errorf("failed to encode png: %w", err)
	}

	return nil
}
