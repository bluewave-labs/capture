package sysfs

import (
	"errors"
	"os/exec"
	"strings"
)

func ShellExec(c string) (string, error) {
	if strings.Contains(c, "&&") || strings.Contains(c, "||") || strings.Contains(c, ";") {
		return "", errors.New("It's forbidden to execute consecutive commands")
	}
	cmd := exec.Command("bash", "-c", c)

	// Run the command and capture the output
	output, err := cmd.Output()

	if err != nil {
		return "", err
	}

	return string(output), nil
}
