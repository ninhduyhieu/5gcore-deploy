package ngapconv

import (
	"github.com/reogac/utils"
	"strconv"
	"strings"
)

func UEAmbrToInt64(modelAmbr string) (v int64, err error) {
	tok := strings.Split(modelAmbr, " ")
	var ambr float64
	if ambr, err = strconv.ParseFloat(tok[0], 64); err != nil {
		err = utils.WrapError("Parse AMBR failed", err)
		return
	} else {
		v = int64(ambr * getUnit(tok[1]))
	}
	return
}

func getUnit(unit string) float64 {
	switch unit {
	case "bps":
		return 1.0
	case "Kbps":
		return 1000.0
	case "Mbps":
		return 1000000.0
	case "Gbps":
		return 1000000000.0
	case "Tbps":
		return 1000000000000.0
	}
	return 1.0
}
