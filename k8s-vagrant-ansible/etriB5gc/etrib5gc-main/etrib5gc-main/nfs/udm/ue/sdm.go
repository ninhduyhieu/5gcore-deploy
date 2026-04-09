package ue

import (
	"context"
	"etrib5gc/internal/datastore"
	udmcontext "etrib5gc/nfs/udm/context"
	"github.com/reogac/sbi/apis/udm/sdm"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func GetAmData(ctx context.Context, supi string, netId *models.PlmnIdNid, supportedFeatures string) (rsp *models.AccessAndMobilitySubscriptionData, err error) {
	ow := datastore.NewObjectLoader[models.AccessAndMobilitySubscriptionData](supi+"_am_data", nil, nil)
	plmnId := udmcontext.PlmnId()
	if netId != nil {
		plmnId = &models.PlmnId{
			Mcc: netId.Mcc,
			Mnc: netId.Mnc,
		}
	}
	rsp, err = datastore.ReadObject(_store, ctx, ow, newUe(supi, plmnId).readAmData)
	return
}

func GetSmfSelData(ctx context.Context, req *sdm.GetSmfSelDataParams) (rsp *models.SmfSelectionSubscriptionData, err error) {
	ow := datastore.NewObjectLoader[models.SmfSelectionSubscriptionData](req.Supi+"_smf_sel", nil, nil)
	//load it from UDR
	plmnId := req.PlmnId
	if plmnId == nil {
		plmnId = udmcontext.PlmnId()
	}
	rsp, err = datastore.ReadObject(_store, ctx, ow, newUe(req.Supi, plmnId).readSmfSel)
	return
}

func GetSmData(ctx context.Context, req *sdm.GetSmDataParams) (rsp *models.SmSubsData, err error) {
	ow := datastore.NewObjectLoader[models.SmSubsData](req.Supi+"_sm_data", nil, nil)
	plmnId := udmcontext.PlmnId()
	rsp, err = datastore.ReadObject(_store, ctx, ow, newUe(req.Supi, plmnId).readSmData)
	return
}

func GetUeCtxInAmfData(ctx context.Context, supi string) (*models.UeContextInAmfData, error) {
	items := []models.AmfInfo{}
	plmnId := udmcontext.PlmnId()

	ow1 := datastore.NewObjectLoader[models.Amf3GppAccessRegistration](supi+"_amf_3gpp", nil, nil)
	if gpp, err := datastore.ReadObject(_store, ctx, ow1, newUe(supi, plmnId).readAmf3GppAccessRegistration); err != nil && err != datastore.NilErr {
		return nil, utils.WrapError("Read Amf3GppAccessRegistartion", err)
	} else if gpp != nil {
		items = append(items, models.AmfInfo{
			AmfInstanceId: gpp.AmfInstanceId,
			AccessType:    models.ACCESSTYPE_3GPP_ACCESS,
			Guami:         gpp.Guami,
		})

	}

	ow2 := datastore.NewObjectLoader[models.AmfNon3GppAccessRegistration](supi+"_amf_non_3gpp", nil, nil)
	if non3Gpp, err := datastore.ReadObject(_store, ctx, ow2, newUe(supi, plmnId).readAmfNon3GppAccessRegistration); err != nil && err != datastore.NilErr {
		return nil, utils.WrapError("Read AmfNon3GppAccessRegistartion", err)
	} else if non3Gpp != nil {
		items = append(items, models.AmfInfo{
			AmfInstanceId: non3Gpp.AmfInstanceId,
			AccessType:    models.ACCESSTYPE_NON_3GPP_ACCESS,
			Guami:         non3Gpp.Guami,
		})

	}

	return &models.UeContextInAmfData{
		AmfInfo: items,
	}, nil
}

func loadSessions(ctx context.Context, supi string) (map[string]models.SmfRegistration, error) {
	ow := datastore.NewObjectLoader[map[string]models.SmfRegistration](supi+"_pdu_sessions", nil, nil)
	plmnId := udmcontext.PlmnId()
	if sessions, err := datastore.ReadObject(_store, ctx, ow, newUe(supi, plmnId).readSmfRegistrations); err != nil {
		return nil, err
	} else {
		return *sessions, nil
	}
}

func writeSessions(ctx context.Context, supi string, sessions map[string]models.SmfRegistration) error {
	ow := datastore.NewObjectWriter(supi+"_pdu_sessions", &sessions, nil)
	plmnId := udmcontext.PlmnId()
	return datastore.WriteObject(_store, ctx, ow, newUe(supi, plmnId).writeSmfRegistrations)
}

func GetUeCtxInSmfData(ctx context.Context, supi string) (*models.UeContextInSmfData, error) {
	if sessions, err := loadSessions(ctx, supi); err != nil {
		return nil, utils.WrapError("Load Smf registrations", err)
	} else {
		items := make(map[string]models.PduSession)
		for id, s := range sessions {
			items[id] = models.PduSession{
				Dnn:           s.Dnn,
				SmfInstanceId: s.SmfInstanceId,
				PlmnId:        s.PlmnId,
				SingleNssai:   &s.SingleNssai,
			}
		}

		return &models.UeContextInSmfData{
			PduSessions: items,
		}, nil
	}
}
