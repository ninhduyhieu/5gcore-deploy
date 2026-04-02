package auth

import (
	"etrib5gc/internal/datastore"
	"etrib5gc/logctx"
	"github.com/sirupsen/logrus"
)

var _store *datastore.DataStore2
var log *logrus.Entry

func CreateDataStore() {
	if _store == nil {
		log = logctx.Entry(logrus.Fields{
			"mod": "auth",
		})
		_store = datastore.CreateDataStore2("ausf", nil, logctx.Entry(nil))
	}
}

func CloseDataStore() {
	if _store != nil {
		_store.Close()
	}
}
