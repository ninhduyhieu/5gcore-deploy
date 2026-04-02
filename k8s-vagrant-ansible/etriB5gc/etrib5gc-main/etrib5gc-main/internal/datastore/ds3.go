package datastore

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	CACHER_TTL   time.Duration = 5 * time.Minute
	CACHER_CLEAN time.Duration = 10 * time.Minute
	MEM_TTL      time.Duration = 30 * time.Minute
)

var NilErr error = fmt.Errorf("object not jound")

type DataStore3[DB any] struct {
	db DB
	*DataStore2
}

func CreateDataStore3[DB any](memPrefix string, mem *redis.Client, db DB, log *logrus.Entry) *DataStore3[DB] {
	return &DataStore3[DB]{
		db:         db,
		DataStore2: CreateDataStore2(memPrefix, mem, log),
	}
}

func ReadObject[T, DB any](ds *DataStore3[DB], ctx context.Context, ow *ObjectWrapper[T], readDb func(DB, context.Context) (*T, error)) (*T, error) {
	if obj, err := ReadObject2(ds.DataStore2, ctx, ow); err != nil && err != NilErr {
		return nil, err
	} else if obj != nil {
		return obj, nil
	}
	if readDb != nil {
		if obj, err := readDb(ds.db, ctx); err != nil {
			return nil, utils.WrapError("Read from database", err)
		} else if obj != nil { //read success
			if ds.log != nil {
				ds.log.Tracef("object is read from DB")
			}
			if ds.mem != nil {
				if err := writeMem(ds.DataStore2, ctx, ow); err != nil {
					return nil, err
				}
			}
			// and write to cacher
			ds.cacher.Set(ow.key, obj, ds.cacherTTL)
			ow.obj = obj
			return obj, nil
		}

	}

	return nil, NilErr
}

func WriteObject[T, DB any](ds *DataStore3[DB], ctx context.Context, ow *ObjectWrapper[T], writeDb func(DB, context.Context, *T) error) error {
	if err := WriteObject2(ds.DataStore2, ctx, ow); err != nil {
		return err
	}
	if writeDb != nil {
		if err := writeDb(ds.db, ctx, ow.obj); err != nil {
			return utils.WrapError("Write to database", err)
		}
	}
	return nil
}

func DeleteObject[DB any](ds *DataStore3[DB], ctx context.Context, key string, deleteDb func(DB) error) error {
	var err1, err2 error

	err1 = ds.Delete(ctx, key)
	//delete from database
	if deleteDb != nil {
		err2 = deleteDb(ds.db)
	}
	if err1 != nil && err2 != nil {
		return fmt.Errorf("Delete object errors: %+v; %+v", err1, err2)
	} else if err1 != nil {
		return utils.WrapError("Delete from redis", err1)
	} else if err2 != nil {
		return utils.WrapError("Delete from database", err2)
	}

	return nil
}
