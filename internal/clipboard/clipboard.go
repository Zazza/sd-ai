package clipboard

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
)

const maxSize = 16 * 1024 * 1024

func ReadImage() (string, error) {
	var data []byte
	var err error

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		data, err = exec.Command("wl-paste", "-t", "image/png").Output()
	} else {
		data, err = exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o").Output()
	}

	if err != nil {
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			return "", fmt.Errorf("failed to read clipboard (install wl-clipboard)")
		}
		return "", fmt.Errorf("failed to read clipboard (install xclip)")
	}

	if len(data) == 0 {
		return "", fmt.Errorf("no image in clipboard")
	}

	if len(data) > maxSize {
		return "", fmt.Errorf("image too large (max 16 MB)")
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
