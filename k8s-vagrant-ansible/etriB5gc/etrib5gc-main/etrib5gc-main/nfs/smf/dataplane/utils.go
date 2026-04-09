package dataplane

import (
	"fmt"
	"github.com/reogac/sbi/models"
	"net"
	"strconv"
	"strings"
)

var standardGbr5QIs = map[int32]struct{}{
	1:  {},
	2:  {},
	3:  {},
	4:  {},
	65: {},
	66: {},
	67: {},
	75: {},
	71: {},
	72: {},
	73: {},
	74: {},
	76: {},
}

func createLinkId(localIp, remoteIp net.IP, direction uint8) string {
	local := "NoIP"
	if len(localIp) > 0 {
		local = localIp.String()
	}

	if direction == DIRECTION_UPLINK {
		remote := "RAN"
		if len(remoteIp) > 0 {
			remote = remoteIp.String()
		}
		return fmt.Sprintf("local=%s: uplink from %s", local, remote)
	}
	remote := "DN"
	if len(remoteIp) > 0 {
		remote = remoteIp.String()
	}

	return fmt.Sprintf("local=%s: downlink-from %s", local, remote)
}

func isGBRFlow(qos *models.QosData) bool {
	if qos.FiveQi != nil {
		_, ok := standardGbr5QIs[int32(*qos.FiveQi)]
		return ok
	}
	return false
}

func BitRateToKbps(bitRate string) uint64 {
	s := strings.Split(bitRate, " ")
	var value int
	var err error

	if len(s) == 0 {
		return 0
	}
	if value, err = strconv.Atoi(s[0]); err != nil {
		return 0
	}

	if len(s) == 1 {
		return uint64(value)
	}

	switch strings.ToLower(s[1]) {
	case "bps":
		return uint64(value / 1000)
	case "kbps":
		return uint64(value)
	case "mbps":
		return uint64(value * 1000)
	case "gbps":
		return uint64(value * 1000000)
	case "tbps":
		return uint64(value * 1000000000)
	}

	return uint64(value)
}

/*
func BitRateToMbps(bitRate string) uint16 {
	s := strings.Split(bitRate, " ")
	var mbps uint16

	var digit int

	if n, err := strconv.Atoi(s[0]); err != nil {
		return 0
	} else {
		digit = n
	}

	switch s[1] {
	case "bps":
		mbps = uint16(digit / 1000000)
	case "Kbps":
		mbps = uint16(digit / 1000)
	case "Mbps":
		mbps = uint16(digit * 1)
	case "Gbps":
		mbps = uint16(digit * 1000)
	case "Tbps":
		mbps = uint16(digit * 1000000)
	}
	return mbps
}
*/
