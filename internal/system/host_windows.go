//go:build windows
// +build windows

package system

import (
	"golang.org/x/sys/windows/registry"
)

func GetPrettyName() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	productName, _, err := key.GetStringValue("ProductName")
	if err != nil {
		return "", err
	}

	return productName, nil
}
