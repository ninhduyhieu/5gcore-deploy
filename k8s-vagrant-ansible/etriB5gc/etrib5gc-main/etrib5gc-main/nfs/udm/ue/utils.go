package ue

import (
	"context"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"strings"
)

func isSupi(supi string) bool {
	parts := strings.Split(supi, "-")
	if len(parts) == 2 && parts[0] == "imsi" {
		return true
	}
	return false
}

type Ue struct {
	supi   string
	plmnId *models.PlmnId
}

func newUe(supi string, plmnId *models.PlmnId) *Ue {
	return &Ue{
		supi:   supi,
		plmnId: plmnId,
	}
}

func (d *Ue) readAuthContext(db BackendDb, ctx context.Context) (*AuthContext, error) {
	if sub, err := db.GetUeAuthSub(ctx, d.supi); err != nil {
		return nil, utils.WrapError("Read subscription data from database", err)
	} else if ueCtx, err := createAuthenticationContext(sub); err != nil {
		return nil, utils.WrapError("Load subscription data", err)
	} else {
		return ueCtx, nil
	}
}

func (d *Ue) writeAuthContext(db BackendDb, ctx context.Context, ueCtx *AuthContext) error {
	ueCtx.sub.SequenceNumber.Sqn = ueCtx.sqn.String()
	log.Infof("Write back authentication subscription with sequence = %s", ueCtx.sqn.String())
	if err := db.SaveUeAuthSub(ctx, ueCtx.sub); err != nil {
		return utils.WrapError("Write subscription data to database", err)
	}
	return nil
}

func (d *Ue) writeAmf3GppAccessRegistration(db BackendDb, ctx context.Context, req *models.Amf3GppAccessRegistration) error {
	if err := db.Write3GppAccessRegistration(ctx, d.supi, d.plmnId, req); err != nil {
		return utils.WrapError("Write 3GppAccessRegistration to database", err)
	}
	return nil
}

func (d *Ue) readAmf3GppAccessRegistration(db BackendDb, ctx context.Context) (*models.Amf3GppAccessRegistration, error) {
	if dat, err := db.Get3GppAccessRegistration(ctx, d.supi, d.plmnId); err != nil {
		return nil, utils.WrapError("Read 3GppAccessRegistration to database", err)
	} else {
		return dat, nil
	}
}
func (d *Ue) writeAmfNon3GppAccessRegistration(db BackendDb, ctx context.Context, req *models.AmfNon3GppAccessRegistration) error {
	if err := db.WriteNon3GppAccessRegistration(ctx, d.supi, d.plmnId, req); err != nil {
		return utils.WrapError("Write Non3GppAccessRegistration to database", err)
	}
	return nil
}

func (d *Ue) readAmfNon3GppAccessRegistration(db BackendDb, ctx context.Context) (*models.AmfNon3GppAccessRegistration, error) {
	if dat, err := db.GetNon3GppAccessRegistration(ctx, d.supi, d.plmnId); err != nil {
		return nil, utils.WrapError("Read Non3GppAccessRegistration to database", err)
	} else {
		return dat, nil
	}
}

func (d *Ue) readSmfRegistrations(db BackendDb, ctx context.Context) (*map[string]models.SmfRegistration, error) {
	if sessions, err := db.GetSmfRegistrations(ctx, d.supi, d.plmnId); err != nil {
		return nil, utils.WrapError("Read SmfRegistration from database", err)
	} else {
		return &sessions, nil
	}
}

func (d *Ue) writeSmfRegistrations(db BackendDb, ctx context.Context, sessions *map[string]models.SmfRegistration) error {
	if err := db.WriteSmfRegistrations(ctx, d.supi, d.plmnId, *sessions); err != nil {
		return utils.WrapError("Write Smf registrations", err)
	}
	return nil
}

func (d *Ue) readAmData(db BackendDb, ctx context.Context) (*models.AccessAndMobilitySubscriptionData, error) {
	if dat, err := db.GetAmData(ctx, d.supi, d.plmnId, ""); err != nil {
		return nil, utils.WrapError("Load AmData from database", err)
	} else {
		return dat, nil
	}
}

func (d *Ue) readSmfSel(db BackendDb, ctx context.Context) (*models.SmfSelectionSubscriptionData, error) {
	if dat, err := db.GetSmfSelSubData(ctx, d.supi, d.plmnId); err != nil {
		return nil, utils.WrapError("Load SmfSelectionSubscriptionData from database", err)
	} else {
		return dat, nil
	}
}

func (d *Ue) readSmData(db BackendDb, ctx context.Context) (*models.SmSubsData, error) {
	if dat, err := db.GetSessManSubData(ctx, d.supi, d.plmnId); err != nil {
		return nil, utils.WrapError("Load SessionManagementSubscriptionData from database", err)
	} else {
		return &models.SmSubsData{
			SessionManagementSubscriptionData: dat,
		}, nil
	}
}
