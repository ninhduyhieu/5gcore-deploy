package common

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
)

// parseHostPort parses host[:port], defaults missing parts.
// Host can be a DNS name, IPv4, or IPv6 (bracketed), or empty.
func ParseHostPort(addr, defaultHost, defaultPort string) (string, int, error) {
	var host, portStr string
	var err error

	if strings.HasPrefix(addr, "[") {
		// IPv6 format like [::1]:443 or [::1]
		end := strings.Index(addr, "]")
		if end == -1 {
			return "", 0, fmt.Errorf("invalid IPv6 address format")
		}
		host = addr[1:end]
		if end+1 < len(addr) && addr[end+1] == ':' {
			portStr = addr[end+2:]
		}
	} else if strings.Contains(addr, ":") {
		// Could be IPv4 or DNS with port
		host, portStr, err = net.SplitHostPort(addr)
		if err != nil {
			// Try "host:" case
			if strings.HasSuffix(addr, ":") {
				host = strings.TrimSuffix(addr, ":")
				portStr = ""
			} else if strings.HasPrefix(addr, ":") {
				host = ""
				portStr = strings.TrimPrefix(addr, ":")
			} else {
				return "", 0, fmt.Errorf("invalid address format: %w", err)
			}
		}
	} else { // Only host or only port
		if _, err := strconv.Atoi(addr); err == nil {
			host = ""
			portStr = addr
		} else {
			host = addr
			portStr = ""
		}
	}

	// Default values
	if host == "" {
		host = defaultHost
	}
	if portStr == "" {
		portStr = defaultPort
	}

	// Validate  port
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port: %s", portStr)
	}

	// Validate host if not IP
	if ip := net.ParseIP(host); ip != nil {
		return host, port, nil
	}
	if !isValidDNSName(host) {
		return "", 0, fmt.Errorf("invalid DNS name: %s", host)
	}

	return host, port, nil
}

// isValidDNSName checks if the input is a valid DNS name (RFC 1035).
func isValidDNSName(s string) bool {
	if len(s) == 0 || len(s) > 253 {
		return false
	}
	labels := strings.Split(s, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if !unicode.IsLetter(rune(label[0])) && !unicode.IsDigit(rune(label[0])) {
			return false
		}
		for _, r := range label {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') {
				return false
			}
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	return true
}
func IsSameNetwork(targetIP net.IP) bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		// Skip down or loopback interfaces
		// if desired
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}

			// Only check same IP version
			if (targetIP.To4() != nil && ipNet.IP.To4() != nil) ||
				(targetIP.To4() == nil && ipNet.IP.To4() == nil) {
				if ipNet.Contains(targetIP) {
					return true
				}
			}
		}
	}

	return false
}

// isWired checks common wired interface naming conventions
func isWired(name string) bool {
	// Wired names often include: "eth", "en", "eno", "ens"
	return strings.HasPrefix(name, "eth") ||
		strings.HasPrefix(name, "en") ||
		strings.HasPrefix(name, "eno") ||
		strings.HasPrefix(name, "ens")
}

// isWireless checks common wireless interface naming conventions
func isWireless(name string) bool {
	// Wireless names often include: "wlan", "wl", "wifi"
	return strings.HasPrefix(name, "wlan") ||
		strings.HasPrefix(name, "wl") ||
		strings.HasPrefix(name, "wifi")
}

// getInterfaceIP gets the first non-loopback IPv4 address of the interface
func getInterfaceIP(iface net.Interface) (net.IP, error) {
	if iface.Flags&net.FlagUp == 0 {
		return nil, fmt.Errorf("interface %s is down", iface.Name)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip := ipNet.IP.To4(); ip != nil {
			return ip, nil
		}
	}

	return nil, fmt.Errorf("no usable IPv4 address found on %s", iface.Name)
}

// getPrimaryIP finds the IP from the first wired interface or falls back to
// wireless
func GetPrimaryIP() (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Try wired interfaces first
	for _, iface := range interfaces {
		if isWired(iface.Name) {
			if ip, err := getInterfaceIP(iface); err == nil {
				return ip, nil
			}
		}
	}

	// Fallback to wireless interfaces
	for _, iface := range interfaces {
		if isWireless(iface.Name) {
			if ip, err := getInterfaceIP(iface); err == nil {
				return ip, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable IP address found on any wired or wireless interface")
}

/*
func GetPublicIP() (ip net.IP) {
	req, err := http.Get("https://ipinfo.io/ip")
	if err != nil {
		log.Errorf("Public IP not discoverable: %+v", err)
		return
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("Public IP not discoverable: %+v", err)
	}
	ip = net.ParseIP(string(body))
	return
}
*/
