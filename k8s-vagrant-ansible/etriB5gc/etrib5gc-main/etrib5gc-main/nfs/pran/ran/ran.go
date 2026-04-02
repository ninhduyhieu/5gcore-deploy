package ran

import (
	"encoding/hex"
	"etrib5gc/common"
	"etrib5gc/nfs/pran/context"
	"etrib5gc/nfs/pran/ue"
	"etrib5gc/util/ngapconv"
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"net"
)

type UePool interface {
	FindByRanNgapId(net.Conn, int64) *ue.UeContext
	FindWithLocalId(int64) *ue.UeContext
	GetRanUePool(net.Conn) []*ue.UeContext
	CreateUeContext(*Ran, int64, sbi.ConsumerClient, bool, *models.UserLocation) *ue.UeContext
}

type Ran struct {
	*logrus.Entry
	access  models.AccessType
	conn    net.Conn
	id      string
	name    string
	ranNets []string
	drx     uint64
	tacs    map[string][]models.Snssai //TACs cover by this RAN
}

type RanUeId struct {
	Conn      net.Conn
	RanNgapId int64
}

func NewRan(conn net.Conn) *Ran {
	ret := &Ran{
		Entry:   log,
		ranNets: context.RanNets(),
		conn:    conn,
		access:  models.ACCESSTYPE_3GPP_ACCESS, //TODO: should be updated from RAN
		tacs:    make(map[string][]models.Snssai),
	}
	return ret
}

func (r *Ran) Access() models.AccessType {
	return r.access
}

func (r *Ran) RanNets() []string {
	return r.ranNets
}

func (r *Ran) Conn() net.Conn {
	return r.conn
}

// write SCTP a message
func (r *Ran) Send(buf []byte) (err error) {
	if len(buf) == 0 {
		err = fmt.Errorf("Attemp to send an empty packet")
		return
	}

	if _, err = r.conn.Write(buf); err != nil {
		err = utils.WrapError("Write packet", err)
	}
	return
}

// signal to close all UeContexts of this RAN
func (r *Ran) removeUes() {
	ue.RemoveUes(r.conn)
}

// signal to close a UeContext
func (r *Ran) removeUe(ranNgapId *int64, cuNgapId *int64) {
	if ranNgapId != nil {
		if ueCtx := ue.FindWithRemoteId(r.conn, *ranNgapId); ueCtx != nil {
			ueCtx.Kill()
		}
	} else if cuNgapId != nil {
		if ueCtx := ue.FindWithLocalId(*cuNgapId); ueCtx != nil {
			ueCtx.Kill()
		}
	}
}
func (r *Ran) updateRanInfo(id *ies.GlobalRANNodeID, name []byte, drx *ies.PagingDRX, taList []ies.SupportedTAItem) (cause *ies.Cause) {
	var plmnId models.PlmnId
	plmnId, r.id, r.access = ranId2String(id)

	r.name = string(name)
	r.drx = uint64(drx.Value)
	for _, item := range taList {
		tac := hex.EncodeToString(item.TAC)
		for _, plmn := range item.BroadcastPLMNList {
			plmnId := ngapconv.PlmnIdToModels(plmn.PLMNIdentity)
			if common.IsPlmnIdEqual(&plmnId, context.PlmnId()) {
				slices := []models.Snssai{}
				for _, snssai := range plmn.TAISliceSupportList {
					slices = append(slices, ngapconv.SNssaiToModels(snssai.SNSSAI))
				}
				r.tacs[tac] = slices
			}
		}
	}

	r.Infof("Update RanInfo: PlmnId: %s; RanId=%s; RanName=%s; TAI list=%v", models.PlmnIdToString(plmnId), r.id, r.name, r.tacs)
	r.Entry = log.WithFields(logrus.Fields{
		"gnb": r.name,
	})

	if !context.CheckSupportedRan(plmnId) {
		cause = &ies.Cause{
			Misc: &ies.CauseMisc{
				Value: ies.CauseMiscUnknownplmn,
			},
			Choice: ies.CausePresentMisc,
		}
	} else {
		context.UpdateTAList(r.tacs)
		_pool.updateRanId(r) //update Ran with new gnB id
	}

	return
}
