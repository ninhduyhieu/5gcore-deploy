package ue

import (
	"context"
	"etrib5gc/internal/datastore"
	"etrib5gc/logctx"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
)

type BackendDb interface {
	GetAmData(context.Context, string, *models.PlmnId, string) (*models.AccessAndMobilitySubscriptionData, error)
	GetUeAuthSub(context.Context, string) (*models.AuthenticationSubscription, error)
	SaveUeAuthSub(context.Context, *models.AuthenticationSubscription) error
	GetSmfSelSubData(context.Context, string, *models.PlmnId) (*models.SmfSelectionSubscriptionData, error)
	GetSessManSubData(context.Context, string, *models.PlmnId) ([]models.SessionManagementSubscriptionData, error)
	Write3GppAccessRegistration(context.Context, string, *models.PlmnId, *models.Amf3GppAccessRegistration) error
	Get3GppAccessRegistration(context.Context, string, *models.PlmnId) (*models.Amf3GppAccessRegistration, error)
	Delete3GppAccessRegistration(context.Context, string, *models.PlmnId) error
	WriteNon3GppAccessRegistration(context.Context, string, *models.PlmnId, *models.AmfNon3GppAccessRegistration) error
	GetNon3GppAccessRegistration(context.Context, string, *models.PlmnId) (*models.AmfNon3GppAccessRegistration, error)
	DeleteNon3GppAccessRegistration(context.Context, string, *models.PlmnId) error
	GetSmfRegistrations(context.Context, string, *models.PlmnId) (map[string]models.SmfRegistration, error)
	WriteSmfRegistrations(context.Context, string, *models.PlmnId, map[string]models.SmfRegistration) error
}

var _store *datastore.DataStore3[BackendDb]
var log *logrus.Entry

func CreateDataStore(db BackendDb) {
	if _store != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "ue",
	})
	_store = datastore.CreateDataStore3[BackendDb]("udm", nil, db, logctx.Entry(nil))
}
func CloseDataStore() {
	if _store != nil {
		_store.Close()
	}
}
