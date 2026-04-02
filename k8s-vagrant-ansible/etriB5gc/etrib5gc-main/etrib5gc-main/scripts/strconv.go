package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func (id *RanUeId) String() string {
	return fmt.Sprintf("%s:%d", id.Ran, id.Id)
}

func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func BoolFromString(str string) (bool, error) {
	switch strings.ToLower(str) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("Not a bool value")
}

func ArrayOfStringToString(list []string) string {
	return strings.Join(list, ",")
}

func ArrayOfStringFromString(str string) (list []string, err error) {
	list = strings.Split(str, ",")
	return
}

func SnssaiToString(snssai Snssai) string {
	return fmt.Sprintf("%d-%s", snssai.Sst, snssai.Sd)
}

func SnssaiFromString(str string) (snssai *Snssai, err error) {
	parts := strings.Split(str, "-")
	if len(parts) == 0 || len(parts) > 2 {
		err = fmt.Errorf("Invalid Snssai string")
		return
	}
	var sst int
	if sst, err = IntFromString(parts[0]); err != nil {
		err = fmt.Errorf("Fail to decode Sst: %+v", err)
		return
	}
	snssai = &Snssai{
		Sst: sst,
	}
	if len(parts) == 2 {
		snssai.Sd = parts[1]
	}
	return
}

func IntToString(v int) string {
	return fmt.Sprintf("%d", v)
}

func IntFromString(str string) (v int, err error) {
	var tmp int64
	if tmp, err = strconv.ParseInt(str, 10, 64); err != nil {
		return
	}
	v = int(tmp)
	return
}

func Int16ToString(v int16) string {
	return fmt.Sprintf("%d", v)
}

func Int16FromString(str string) (v int16, err error) {
	var tmp int64
	if tmp, err = strconv.ParseInt(str, 10, 16); err != nil {
		return
	}
	v = int16(tmp)
	return
}

func Int32ToString(v int16) string {
	return fmt.Sprintf("%d", v)
}

func Int32FromString(str string) (v int32, err error) {
	var tmp int64
	if tmp, err = strconv.ParseInt(str, 10, 32); err != nil {
		return
	}
	v = int32(tmp)
	return
}

func Int64ToString(v int64) string {
	return fmt.Sprintf("%d", v)
}

func Int64FromString(str string) (v int64, err error) {
	v, err = strconv.ParseInt(str, 10, 64)
	return
}

func PlmnIdToString(id PlmnId) string {
	return fmt.Sprintf("%s-%s", id.Mcc, id.Mnc)
}

func PlmnIdFromString(str string) (id *PlmnId, err error) {
	if parts := strings.Split(str, "-"); len(parts) != 2 {
		err = fmt.Errorf("Invalid PlmnId: %s", str)
	} else {
		id = &PlmnId{
			Mcc: parts[0],
			Mnc: parts[1],
		}
		//check validity
		_, err = PlmnId2Bytes(id)
	}
	return
}

func PlmnIdFromPlmnIdNid(nid *PlmnIdNid) *PlmnId {
	if nid == nil {
		return nil
	}
	return &PlmnId{
		Mcc: nid.Mcc,
		Mnc: nid.Mnc,
	}
}

func PlmnIdNidToString(id PlmnIdNid) string {
	return fmt.Sprintf("%s-%s-%s", id.Mcc, id.Mnc, id.Nid)
}
func PlmnIdNidFromString(str string) (id *PlmnIdNid, err error) {
	if parts := strings.Split(str, "-"); len(parts) != 3 {
		err = fmt.Errorf("Invalid PlmnIdNid: %s", str)
	} else {
		id = &PlmnIdNid{
			Mcc: parts[0],
			Mnc: parts[1],
			Nid: parts[2],
		}
	}
	return
}
func AppPortIdToString(id AppPortId) string {
	p1 := ""
	p2 := ""
	if id.OriginatorPort != nil {
		p1 = IntToString(*id.OriginatorPort)
	}
	if id.DestinationPort != nil {
		p2 = IntToString(*id.DestinationPort)
	}
	return fmt.Sprintf("%s:%s", p1, p2)
}

func AppPortIdFromString(str string) (id *AppPortId, err error) {
	parts := strings.Split(str, ":")
	if len(parts) > 2 {
		err = fmt.Errorf("Invalid AppPortId: %s", str)
		return
	}
	var tmpId AppPortId
	if len(parts) >= 1 {
		tmpId.OriginatorPort = new(int)
		if *tmpId.OriginatorPort, err = IntFromString(parts[0]); err != nil {
			err = fmt.Errorf("Parse OriginatorPort failed: %+v", err)
			return
		}
	}
	if len(parts) >= 2 {
		tmpId.DestinationPort = new(int)
		if *tmpId.DestinationPort, err = IntFromString(parts[1]); err != nil {
			err = fmt.Errorf("Parse DestinationPort failed: %+v", err)
			return
		}
	}
	id = &tmpId
	return
}

func ArrayOfPlmnIdToString(list []PlmnId) string {
	strList := []string{}
	for _, item := range list {
		strList = append(strList, PlmnIdToString(item))
	}
	return strings.Join(strList, ",")
}

func ArrayOfPlmnIdFromString(str string) (list []PlmnId, err error) {
	parts := strings.Split(str, ",")
	var plmnId *PlmnId
	for _, part := range parts {
		if plmnId, err = PlmnIdFromString(part); err != nil {
			err = fmt.Errorf("Parse PlmnId %s failed: %+v", part, err)
			return
		}
		list = append(list, *plmnId)
	}

	return
}

func EndpointInfoToString(ep EndpointInfo) string {
	buf, _ := json.Marshal(&ep)
	return string(buf)
}

func EndpointInfoFromString(str string) (id *EndpointInfo, err error) {
	var tmpId EndpointInfo
	if err = json.Unmarshal([]byte(str), &tmpId); err != nil {
		err = fmt.Errorf("Json unmarshal EndpointInfo failed: %+v", err)
		return
	}
	id = &tmpId
	return
}
