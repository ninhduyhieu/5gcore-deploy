package models

import (
	"encoding/binary"
	"encoding/json"
	"etrib5gc/nas"
	"fmt"
	"strings"

	"encoding/hex"
	"strconv"
)

type SubscribedSnssai struct {
	SubscribedSnssai  Snssai `json:"subscribedSnssai"`
	DefaultIndication bool   `json:"defaultIndication,omitempty"`
}

func (prob *ProblemDetails) Error() string {
	return fmt.Sprintf("Status: %d, Detail: %s", prob.Status, prob.Detail)
}

func CreateProblemDetails(status int, detail string) *ProblemDetails {
	return &ProblemDetails{
		Status: status,
		Detail: detail,
	}
}
func StatusFromExtProblemDetails(info *ExtProblemDetails) int {
	return 500
}
func StatusFromPostPduSessionsErrorResponse(info *PostPduSessionsErrorResponse) int {
	return 500
}
func StatusFromUeContextCreateError(info *UeContextCreateError) int {
	return 500
}
func StatusFromN2InformationTransferError(info *N2InformationTransferError) int {
	return 500
}
func StatusFromN1N2MessageTransferError(info *N1N2MessageTransferError) int {
	return 500
}
func StatusFromAssignEbiError(info *AssignEbiError) int {
	return 500
}
func StatusFromUpdateSmContextErrorResponse(info *UpdateSmContextErrorResponse) int {
	return 500
}
func StatusFromPostSmContextsErrorResponse(info *PostSmContextsErrorResponse) int {
	return 500
}
func StatusFromUpdatePduSessionErrorResponse(info *UpdatePduSessionErrorResponse) int {
	return 500
}

func (ngksi *NgKsi) String() string {
	if ngksi.Tsc == SCTYPE_NATIVE {
		return fmt.Sprintf("native:%d", ngksi.Ksi)
	} else {
		return fmt.Sprintf("mapped:%d", ngksi.Ksi)
	}
}

func (ngksi *NgKsi) NasType() *nas.KeySetIdentifier {
	ret := &nas.KeySetIdentifier{
		Id: uint8(ngksi.Ksi),
	}
	ret.Tsc = nas.TscMapped
	if ngksi.Tsc == SCTYPE_NATIVE {
		ret.Tsc = nas.TscNative
	}
	return ret
}

func (id *PlmnId) NasType() *nas.PlmnId {
	nasPlmnId := new(nas.PlmnId)
	if nasPlmnId.Set(id.Mcc, id.Mnc) != nil {
		return nasPlmnId
	}
	return nil
}

func (id *PlmnId) Equal(peer *PlmnId) bool {
	return id.Mcc == peer.Mcc && id.Mnc == peer.Mnc
}

func (id *PlmnId) UnmarshalJSON(b []byte) (err error) {
	tmpid := struct {
		Mcc string
		Mnc string
	}{}

	if err = json.Unmarshal(b, &tmpid); err != nil {
		return
	}
	id.Mnc = tmpid.Mnc
	id.Mcc = tmpid.Mcc

	if _, err = PlmnId2Bytes(id); err != nil {
		return
	}
	return
}

func PlmnId2Bytes(id *PlmnId) (buf []uint8, err error) {
	if len(id.Mcc) != 3 {
		err = fmt.Errorf("Mcc len must be 3: %s", id.Mcc)
		return
	}
	if len(id.Mnc) != 2 && len(id.Mnc) != 3 {
		err = fmt.Errorf("Mnc len must be 2 or 3: %s", id.Mnc)
		return
	}

	var (
		mcc [3]uint8
		mnc [3]uint8
		tmp int
	)

	mnc[2] = 0x0f

	for i := 0; i < 3; i++ {
		if tmp, err = strconv.Atoi(string(id.Mcc[i])); err != nil {
			return
		}
		mcc[i] = uint8(tmp)
	}
	for i := 0; i < len(id.Mnc); i++ {
		if tmp, err = strconv.Atoi(string(id.Mnc[i])); err != nil {
			return
		}
		mnc[i] = uint8(tmp)
	}

	buf = []uint8{
		(mcc[1] << 4) | mcc[0],
		(mnc[2] << 4) | mcc[2],
		(mnc[1] << 4) | mnc[0],
	}
	return
}

func Bytes2PlmnId(buf []byte) (id *PlmnId, err error) {
	if len(buf) != 3 {
		err = fmt.Errorf("plmnid must be 3-byte length")
		return
	}
	var mcc [3]byte
	var mnc [3]byte
	mcc[0] = buf[0] & 0x0f
	mcc[1] = (buf[0] & 0xf0) >> 4
	mcc[2] = (buf[1] & 0x0f)

	mnc[0] = (buf[2] & 0x0f)
	mnc[1] = (buf[2] & 0xf0) >> 4
	mnc[2] = (buf[1] & 0xf0) >> 4

	tmp := []byte{(mcc[0] << 4) | mcc[1], (mcc[2] << 4) | mnc[0], (mnc[1] << 4) | mnc[2]}

	str := hex.EncodeToString(tmp)
	plmnid := PlmnId{
		Mcc: str[:3],
	}
	if str[5] == 'f' {
		plmnid.Mnc = str[3:5] //discard the last letter
	} else {
		plmnid.Mnc = str[3:]
	}
	id = &plmnid
	return
}

func (id *Snssai) NasType() (nasType *nas.SNssai) {
	nasType = new(nas.SNssai)
	nasType.Set(uint8(id.Sst), id.Sd) //NOTE: ignore error if Sd is invalid
	return
}

func (id Snssai) String() string {
	return fmt.Sprintf("%d-%s", id.Sst, id.Sd)
}

func SlicesEqual(s1 Snssai, s2 Snssai) bool {
	return s1.Sst == s2.Sst && strings.Compare(s1.Sd, s2.Sd) == 0
}
func AmbrFromNas(nasAmbr *nas.SessionAmbr) Ambr {
	//TODO: conver from nas ambr
	return Ambr{}
}

func (ambr *Ambr) NasSessionAmbr() *nas.SessionAmbr {
	ulUnit, ulRate := unitAndRate(ambr.Uplink)
	dlUnit, dlRate := unitAndRate(ambr.Downlink)
	return nas.NewSessionAmbr(ulUnit, ulRate, dlUnit, dlRate)
}

func unitAndRate(link string) (u uint8, r uint16) {
	parts := strings.Split(link, " ")
	tmp, _ := strconv.ParseUint(parts[0], 10, 16)
	r = uint16(tmp)
	if len(parts) > 1 {
		u = str2AmbrUnit(parts[1])
	} else {
		u = nas.SessionAMBRUnitNotUsed
	}
	return
}

func str2AmbrUnit(unit string) uint8 {
	switch unit {
	case "bps":
		return nas.SessionAMBRUnitNotUsed
	case "Kbps":
		return nas.SessionAMBRUnit1Kbps
	case "Mbps":
		return nas.SessionAMBRUnit1Mbps
	case "Gbps":
		return nas.SessionAMBRUnit1Gbps
	case "Tbps":
		return nas.SessionAMBRUnit1Tbps
	case "Pbps":
		return nas.SessionAMBRUnit1Pbps
	}
	return nas.SessionAMBRUnitNotUsed
}

func (tai *Tai) NasType() *nas.TrackingAreaIdentity {
	nasTai := new(nas.TrackingAreaIdentity)
	if nasTai.Set(tai.PlmnId.Mcc, tai.PlmnId.Mnc, tai.Tac) == nil {
		return nasTai
	}
	return nil
}

func TaiFromNasTai(nasTai *nas.TrackingAreaIdentity) (t Tai) {
	t.PlmnId.Mcc, t.PlmnId.Mnc = nasTai.PlmnId.Get()
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], nasTai.Tac)
	t.Tac = hex.EncodeToString(buf[1:])
	return
}
