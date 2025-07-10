//go:build linux
// +build linux

package metric

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

func GetPrettyName() (string, error) {
	const osReleasePath = "/etc/os-release"

	file, err := os.Open(osReleasePath)
	if err != nil {
		return "", errors.New("unable to open /etc/os-release: " + err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			pretty := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(pretty, `"`), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", errors.New("error reading /etc/os-release: " + err.Error())
	}

	return "", errors.New("PRETTY_NAME field not found in /etc/os-release")
}
