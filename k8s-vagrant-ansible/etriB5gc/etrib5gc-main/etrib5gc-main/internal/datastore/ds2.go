package datastore

import (
	"context"
	//	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"time"
)

type DataStore2 struct {
	log          *logrus.Entry
	mem          *redis.Client
	invalidateCh string
	memPrefix    string
	cacher       Cacher
	cacherTTL    time.Duration
	memTTL       time.Duration
	done         chan struct{}
}

func CreateDataStore2(memPrefix string, mem *redis.Client, log *logrus.Entry) *DataStore2 {
	ds := &DataStore2{
		cacher:       NewGoCache(CACHER_TTL, CACHER_CLEAN),
		cacherTTL:    CACHER_TTL,
		mem:          mem,
		memTTL:       MEM_TTL,
		memPrefix:    memPrefix,
		invalidateCh: memPrefix + ":invalidate",
		done:         make(chan struct{}),
	}
	if log != nil {
		ds.log = log.WithFields(
			logrus.Fields{
				"mod": "datastore",
			},
		)
	}
	if mem != nil {
		pubsub := ds.mem.Subscribe(context.Background(), ds.invalidateCh)
		ch := pubsub.Channel()
		go func() {
			defer pubsub.Close()
			for {
				select {
				case msg := <-ch:
					ds.cacher.Delete(msg.Payload)
				case <-ds.done:
					return
				}
			}
		}()
	}
	return ds
}

func (ds *DataStore2) Close() {
	if ds.mem != nil {
		ds.done <- struct{}{}
	}
}

func ReadObject2[T any](ds *DataStore2, ctx context.Context, ow *ObjectWrapper[T]) (*T, error) {
	useRedis := ds.mem != nil
	//1. Try cacher
	if val, ok := ds.cacher.Get(ow.key); !ok {
		if useRedis {
			// 2. Try Redis
			key := ds.memPrefix + ow.key
			if val, err := ds.mem.Get(ctx, key).Result(); err == nil { //found in redis
				var obj T
				if err := ow.decode([]byte(val), &obj); err != nil {
					return nil, utils.WrapError("Decode redis object", err)
				} else {
					ds.cacher.Set(ow.key, &obj, ds.cacherTTL)
					return &obj, nil
				}
			} else if err != redis.Nil { //redis hard error
				return nil, utils.WrapError("Read Redis", err)
			}
		}
	} else { //found in cacher
		if ds.log != nil {
			ds.log.Tracef("Object is found from the cacher")
		}
		obj, _ := val.(*T)
		ow.obj = obj
		return obj, nil
	}
	return nil, NilErr
}

func WriteObject2[T any](ds *DataStore2, ctx context.Context, ow *ObjectWrapper[T]) error {
	//write cacher (reset TTL if it was there)
	ds.cacher.Set(ow.key, ow.obj, ds.cacherTTL)

	useRedis := ds.mem != nil
	if useRedis {
		if err := writeMem(ds, ctx, ow); err != nil {
			return err
		}
	}
	return nil
}

func writeMem[T any](ds *DataStore2, ctx context.Context, ow *ObjectWrapper[T]) error {
	//write to redis
	key := ds.memPrefix + ow.key
	if dat, err := ow.encode(ow.obj); err != nil {
		return utils.WrapError("Encode to write to redis", err)
	} else if err := ds.mem.Set(ctx, key, dat, ds.memTTL).Err(); err != nil {
		return utils.WrapError("Write to redis", err)
	}
	//invalidate redis for lazy load by other peer-cachers
	if err := ds.mem.Publish(ctx, ds.invalidateCh, ow.key).Err(); err != nil {
		return utils.WrapError("Redis invalidate", err)
	}
	return nil
}

func (ds *DataStore2) Delete(ctx context.Context, key string) error {
	//delete from cacher
	ds.cacher.Delete(key)
	//delete from redis
	if ds.mem != nil {
		if err := ds.mem.Del(ctx, ds.memPrefix+key).Err(); err != nil {
			return utils.WrapError("Delete from redis", err)
		}
	}
	return nil
}
