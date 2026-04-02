package udr

import (
	"github.com/reogac/sbi/models"
)

type MongoDoc[T any] struct {
	UeId   string `bson:"ueId"`
	PlmnId string `bson:"plmnId"`
	ID     string `bson:"_id"`
	Dat    T      `bson:"dat:`
}

func newMongoDoc[T any](ueId string, plmnId *models.PlmnId, obj *T) *MongoDoc[T] {
	id := ueId
	plmnIdStr := ""
	if plmnId != nil { //SUPI
		plmnIdStr = models.PlmnIdToString(*plmnId)
		id = ueId + "_" + plmnIdStr
	}

	doc := &MongoDoc[T]{
		UeId:   ueId,
		PlmnId: plmnIdStr,
		ID:     id,
	}
	if obj != nil {
		doc.Dat = *obj
	}
	return doc
}
