//go:build darwin
// +build darwin

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func GetPrettyName() (string, error) {
	cmd := exec.Command("sw_vers")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	output := out.String()

	var productName, productVersion string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if trimmed, found := strings.CutPrefix(line, "ProductName:"); found {
			productName = strings.TrimSpace(trimmed)
		} else if trimmed, found := strings.CutPrefix(line, "ProductVersion:"); found {
			productVersion = strings.TrimSpace(trimmed)
		}
	}

	if productName != "" && productVersion != "" {
		return fmt.Sprintf("%s %s", productName, productVersion), nil
	}

	return "", fmt.Errorf("could not determine macOS version")
}
