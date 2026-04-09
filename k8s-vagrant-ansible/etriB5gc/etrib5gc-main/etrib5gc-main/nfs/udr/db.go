package udr

import (
	"context"
	"etrib5gc/logctx"
	"fmt"
	"github.com/reogac/sbi/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	MONGO_CONNECT_TIMEOUT time.Duration = 500 * time.Millisecond
	MONGO_QUERY_TIMEOUT   time.Duration = 500 * time.Millisecond
)

const (
	GPP_REG_CTX_COL     string = "gpp_reg_ctx"
	NON_GPP_REG_CTX_COL string = "non_gpp_reg_ctx"
	AUTH_CTX_COL        string = "auth_ctx"
	SESSION_CTX_COL     string = "pdu_sessions"
)

type BackendDb struct {
	*logrus.Entry
	config *models.UdrConfiguration
	client *mongo.Client
}

func New(cfg *models.UdrConfiguration) *BackendDb {
	db := &BackendDb{
		config: cfg,
		Entry: logctx.Entry(logrus.Fields{
			"mod": "udr",
		}),
	}

	return db
}

func (db *BackendDb) Connect(init bool) (err error) {
	if db.client != nil {
		return
	}

	if db.config.Mock == nil || !(*db.config.Mock) {
		//		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		opts := options.Client().ApplyURI(db.config.Url) //.SetServerAPIOptions(serverAPI)
		ctx, cancel := context.WithTimeout(context.Background(), MONGO_CONNECT_TIMEOUT)
		defer cancel()
		if db.client, err = mongo.Connect(ctx, opts); err != nil {
			return fmt.Errorf("Fail to connect to mongodb: %+v", err)
		}
		if init {
			db.dropCollections()
			err = db.createCollections()
		}
	}

	return
}

func (db *BackendDb) dropCollections() {
	if err := db.client.Database(db.config.DbName).Collection(db.config.AuthSub).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", db.config.AuthSub, err)
	}
	if err := db.client.Database(db.config.DbName).Collection(db.config.AmSub).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", db.config.AmSub, err)
	}
	if err := db.client.Database(db.config.DbName).Collection(db.config.SmSub).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", db.config.SmSub, err)
	}
	if err := db.client.Database(db.config.DbName).Collection(db.config.SmfSel).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", db.config.SmfSel, err)
	}

	if err := db.client.Database(db.config.DbName).Collection(SESSION_CTX_COL).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", SESSION_CTX_COL, err)
	}
	if err := db.client.Database(db.config.DbName).Collection(AUTH_CTX_COL).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", AUTH_CTX_COL, err)
	}
	if err := db.client.Database(db.config.DbName).Collection(GPP_REG_CTX_COL).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", GPP_REG_CTX_COL, err)
	}
	if err := db.client.Database(db.config.DbName).Collection(NON_GPP_REG_CTX_COL).Drop(nil); err != nil {
		db.Warnf("Fail to drop collection %s: %+v", NON_GPP_REG_CTX_COL, err)
	}

}

func (db *BackendDb) deleteDoc(ctx context.Context, ueId string, plmnId *models.PlmnId, col string) error {
	coll := db.client.Database(db.config.DbName).Collection(col)
	filter := bson.M{"ueId": ueId, "plmnId": models.PlmnIdToString(*plmnId)}
	_, err := coll.DeleteOne(ctx, filter)
	return err
}

func (db *BackendDb) createCollections() error {
	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), db.config.AuthSub); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", db.config.AuthSub, err)
	}
	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), db.config.AmSub); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", db.config.AmSub, err)
	}
	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), db.config.SmSub); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", db.config.SmSub, err)
	}
	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), db.config.SmfSel); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", db.config.SmfSel, err)
	}

	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), AUTH_CTX_COL); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", AUTH_CTX_COL, err)
	}

	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), SESSION_CTX_COL); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", SESSION_CTX_COL, err)
	}

	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), GPP_REG_CTX_COL); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", GPP_REG_CTX_COL, err)
	}

	if err := db.client.Database(db.config.DbName).CreateCollection(context.TODO(), NON_GPP_REG_CTX_COL); err != nil {
		return fmt.Errorf("Fail to create collection %s: %+v", NON_GPP_REG_CTX_COL, err)
	}

	db.Infof("All collections created")
	return nil
}

func (db *BackendDb) Close() {
	if db.client == nil {
		return
	}

	if err := db.client.Disconnect(context.TODO()); err != nil {
		db.Errorf("Disconnect to MongoDB failed: %+v", err)
	}
	db.Infof("Connection to MongoDB closed.")
	return
}

func (db *BackendDb) GetUeAuthSub(ctx context.Context, supi string) (sub *models.AuthenticationSubscription, err error) {
	if db.config.Mock != nil && *db.config.Mock {
		//return a default UE's authentication subscription data
		sub = &models.AuthenticationSubscription{}
		sub.AuthenticationMethod = models.AUTHMETHOD_5G_AKA
		// sub.AuthenticationMethod = models.AUTHMETHOD_EAP_AKA_PRIME
		sub.AuthenticationManagementField = "8000"
		sub.EncPermanentKey = "8baf473f2f8fd09487cccbd7097c6862"
		sub.EncOpcKey = "8e27b6af0e692e750f32667a3b14605d"
		sub.Supi = "imsi-208930000000003"
		sub.SequenceNumber = &models.SequenceNumber{
			Sqn: "0000000000af",
		}
		return sub, nil
	} else {
		return readDoc[models.AuthenticationSubscription](db, ctx, supi, nil, db.config.AuthSub)
	}
}

func (db *BackendDb) SaveUeAuthSub(ctx context.Context, dat *models.AuthenticationSubscription) error {
	return writeDoc[models.AuthenticationSubscription](db, ctx, dat.Supi, nil, dat, db.config.AuthSub, true)
}

func (db *BackendDb) SaveUeAmSub(ctx context.Context, supi string, plmnId *models.PlmnId, dat *models.AccessAndMobilitySubscriptionData) error {
	return writeDoc[models.AccessAndMobilitySubscriptionData](db, ctx, supi, plmnId, dat, db.config.AmSub, false)
}

func (db *BackendDb) SaveUeSmSub(ctx context.Context, supi string, plmnId *models.PlmnId, dat *models.SmSubsData) error {
	return writeDoc[models.SmSubsData](db, ctx, supi, plmnId, dat, db.config.SmSub, false)
}

func (db *BackendDb) SaveUeSmfSel(ctx context.Context, supi string, plmnId *models.PlmnId, dat *models.SmfSelectionSubscriptionData) error {
	return writeDoc[models.SmfSelectionSubscriptionData](db, ctx, supi, plmnId, dat, db.config.SmfSel, false)
}

// for now just use supi, ignore other parameters
func (db *BackendDb) GetAmData(ctx context.Context, supi string, plmnId *models.PlmnId, supportedFeatures string) (*models.AccessAndMobilitySubscriptionData, error) {
	return readDoc[models.AccessAndMobilitySubscriptionData](db, ctx, supi, plmnId, db.config.AmSub)
}

func (db *BackendDb) GetSmfSelSubData(ctx context.Context, supi string, plmnId *models.PlmnId) (*models.SmfSelectionSubscriptionData, error) {
	return readDoc[models.SmfSelectionSubscriptionData](db, ctx, supi, plmnId, db.config.SmfSel)
}

func (db *BackendDb) GetSessManSubData(ctx context.Context, supi string, plmnId *models.PlmnId) (rsp []models.SessionManagementSubscriptionData, err error) {
	var dat *models.SmSubsData
	if dat, err = readDoc[models.SmSubsData](db, ctx, supi, plmnId, db.config.SmSub); err == nil {
		rsp = dat.SessionManagementSubscriptionData
	}
	return
}

func (db *BackendDb) GetSmPolicyData(supi string) (rsp *models.SmPolicyData, err error) {
	//TODO: get session management policy subscription data from udr
	rsp = &models.SmPolicyData{}
	return
}

func (db *BackendDb) Write3GppAccessRegistration(ctx context.Context, supi string, plmnId *models.PlmnId, dat *models.Amf3GppAccessRegistration) error {
	return writeDoc[models.Amf3GppAccessRegistration](db, ctx, supi, plmnId, dat, GPP_REG_CTX_COL, true)
}

func (db *BackendDb) Get3GppAccessRegistration(ctx context.Context, supi string, plmnId *models.PlmnId) (*models.Amf3GppAccessRegistration, error) {
	if dat, err := readDoc[models.Amf3GppAccessRegistration](db, ctx, supi, plmnId, GPP_REG_CTX_COL); err != nil && err == mongo.ErrNoDocuments {
		return nil, nil
	} else {
		return dat, err
	}
}

func (db *BackendDb) Delete3GppAccessRegistration(ctx context.Context, supi string, plmnId *models.PlmnId) error {
	return db.deleteDoc(ctx, supi, plmnId, GPP_REG_CTX_COL)
}
func (db *BackendDb) WriteNon3GppAccessRegistration(ctx context.Context, supi string, plmnId *models.PlmnId, dat *models.AmfNon3GppAccessRegistration) error {
	return writeDoc[models.AmfNon3GppAccessRegistration](db, ctx, supi, plmnId, dat, NON_GPP_REG_CTX_COL, true)
}

func (db *BackendDb) GetNon3GppAccessRegistration(ctx context.Context, supi string, plmnId *models.PlmnId) (*models.AmfNon3GppAccessRegistration, error) {
	if dat, err := readDoc[models.AmfNon3GppAccessRegistration](db, ctx, supi, plmnId, NON_GPP_REG_CTX_COL); err != nil && err == mongo.ErrNoDocuments {
		return nil, nil
	} else {
		return dat, err
	}
}

func (db *BackendDb) DeleteNon3GppAccessRegistration(ctx context.Context, supi string, plmnId *models.PlmnId) error {
	return db.deleteDoc(ctx, supi, plmnId, NON_GPP_REG_CTX_COL)
}

func (db *BackendDb) GetSmfRegistrations(ctx context.Context, supi string, plmnId *models.PlmnId) (map[string]models.SmfRegistration, error) {
	if dat, err := readDoc[map[string]models.SmfRegistration](db, ctx, supi, plmnId, SESSION_CTX_COL); err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	} else if dat != nil {
		return *dat, nil
	}
	return nil, nil
}

func (db *BackendDb) WriteSmfRegistrations(ctx context.Context, supi string, plmnId *models.PlmnId, sessions map[string]models.SmfRegistration) error {
	return writeDoc[map[string]models.SmfRegistration](db, ctx, supi, plmnId, &sessions, SESSION_CTX_COL, true)
}
