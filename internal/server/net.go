package server

import "net"

// GetLocalIP retrieves the local IP address of the machine.
// It returns the first non-loopback IPv4 address found.
// If no valid address is found, it returns "<ip-address>" as a placeholder.
// This function is used to display the local IP address in the log message.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "<ip-address>"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "<ip-address>"
}
