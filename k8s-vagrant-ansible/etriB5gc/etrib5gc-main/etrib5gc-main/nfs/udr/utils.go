package udr

import (
	"context"
	"github.com/reogac/sbi/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func writeDoc[T any](db *BackendDb, ctx context.Context, ueId string, plmnId *models.PlmnId, doc *T, col string, replace bool) error {
	mongoDoc := newMongoDoc[T](ueId, plmnId, doc)
	coll := db.client.Database(db.config.DbName).Collection(col)
	if replace {
		opts := options.Replace().SetUpsert(true)
		if _, err := coll.ReplaceOne(ctx, bson.M{"_id": mongoDoc.ID}, mongoDoc, opts); err != nil {
			return err
		}
	} else {
		if _, err := coll.InsertOne(ctx, mongoDoc); err != nil {
			return err
		}
	}

	return nil
}
func readDoc[T any](db *BackendDb, ctx context.Context, ueId string, plmnId *models.PlmnId, col string) (*T, error) {
	doc := newMongoDoc[T](ueId, plmnId, nil)
	coll := db.client.Database(db.config.DbName).Collection(col)
	id := ueId

	if plmnId != nil {
		id = ueId + "_" + models.PlmnIdToString(*plmnId)
	}
	filter := bson.M{"_id": id}
	if err := coll.FindOne(ctx, filter).Decode(doc); err != nil {
		return nil, err
	}

	return &doc.Dat, nil
}
