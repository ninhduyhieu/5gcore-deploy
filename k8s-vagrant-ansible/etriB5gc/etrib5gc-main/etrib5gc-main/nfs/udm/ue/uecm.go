package ue

import (
	"context"
	"etrib5gc/internal/datastore"
	udmcontext "etrib5gc/nfs/udm/context"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func Create3GppRegistration(ctx context.Context, supi string, msg *models.Amf3GppAccessRegistration) error {
	ow := datastore.NewObjectWriter(supi+"_amf_3gpp", msg, nil)
	plmnId := udmcontext.PlmnId()
	return datastore.WriteObject(_store, ctx, ow, newUe(supi, plmnId).writeAmf3GppAccessRegistration)
}

func CreateNon3GppRegistration(ctx context.Context, supi string, msg *models.AmfNon3GppAccessRegistration) error {
	ow := datastore.NewObjectWriter(supi+"_amf_non_3gpp", msg, nil)
	plmnId := udmcontext.PlmnId()
	return datastore.WriteObject(_store, ctx, ow, newUe(supi, plmnId).writeAmfNon3GppAccessRegistration)
}

func DeregisterUe(ctx context.Context, supi string) (err error) {
	deleteDb := func(db BackendDb) error {
		return db.Delete3GppAccessRegistration(ctx, supi, udmcontext.PlmnId())
	}
	return datastore.DeleteObject(_store, ctx, supi+"_amf_3gpp", deleteDb)
}

func RegisterPduSession(ctx context.Context, supi string, sessionId int, req *models.SmfRegistration) (*models.SmfRegistration, error) {
	sessions, err := loadSessions(ctx, supi)
	if err != nil {
		return nil, utils.WrapError("Load existing sessions from database", err)
	}
	idStr := fmt.Sprintf("%d", sessionId)
	if s, ok := sessions[idStr]; ok {
		//TODO:check and release existing Pdu session
		_ = s
	}
	sessions[idStr] = *req
	if err := writeSessions(ctx, supi, sessions); err != nil {
		return nil, utils.WrapError("Write sessions to database", err)
	}

	return req, nil
}

func DeregisterPduSession(ctx context.Context, supi string, sessionId int) error {
	sessions, err := loadSessions(ctx, supi)
	if err != nil {
		return utils.WrapError("Load existing sessions from database", err)
	}
	idStr := fmt.Sprintf("%d", sessionId)
	if _, ok := sessions[idStr]; ok {
		delete(sessions, idStr)
		if err := writeSessions(ctx, supi, sessions); err != nil {
			return utils.WrapError("Write sessions to database", err)
		}
	} else {
		return fmt.Errorf("Session not found")
	}
	return nil
}
