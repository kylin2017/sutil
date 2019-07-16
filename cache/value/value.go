// Copyright 2014 The mqrouter Author. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/shawnfeng/sutil/cache"
	"github.com/shawnfeng/sutil/cache/redis"
	"github.com/shawnfeng/sutil/slog/slog"
	"strings"
	"time"
)

// key类型只支持int（包含有无符号，8，16，32，64位）和string
type LoadFunc func(key interface{}) (value interface{}, err error)

type Cache struct {
	namespace string
	prefix    string
	load      LoadFunc
	expire    time.Duration
}

func NewCache(namespace, prefix string, expire time.Duration, load LoadFunc) *Cache {
	return &Cache{
		namespace: strings.Replace(namespace, "/", ".", -1),
		prefix:    prefix,
		load:      load,
		expire:    expire,
	}
}

func (m *Cache) Get(ctx context.Context, key, value interface{}) error {
	fun := "Cache.Get -->"

	span, ctx := opentracing.StartSpanFromContext(ctx, "cache.value.Get")
	defer span.Finish()

	err := m.getValueFromCache(ctx, key, value)
	if err == nil {
		return nil
	}

	if err.Error() != redis.RedisNil {
		slog.Errorf(ctx, "%s cache key: %v err: %v", fun, key, err)
		return fmt.Errorf("%s cache key: %v err: %v", fun, key, err)
	}

	slog.Infof(ctx, "%s miss key: %v, err: %s", fun, key, err)

	err = m.loadValueToCache(ctx, key)
	if err != nil {
		slog.Errorf(ctx, "%s loadValueToCache key: %v err: %v", fun, key, err)
		return err
	}

	//简单处理interface对象构造的问题
	return m.getValueFromCache(ctx, key, value)
}

func (m *Cache) Del(ctx context.Context, key interface{}) error {
	fun := "Cache.Del -->"

	span, ctx := opentracing.StartSpanFromContext(ctx, "cache.value.Del")
	defer span.Finish()

	skey, err := m.fixKey(key)
	if err != nil {
		slog.Errorf(ctx, "%s fixkey, key: %v err: %v", fun, key, err)
		return err
	}

	client := redis.DefaultInstanceManager.GetInstance(ctx, m.namespace)
	if client == nil {
		slog.Errorf(ctx, "%s get instance err, namespace: %s", fun, m.namespace)
		return fmt.Errorf("get instance err, namespace: %s", m.namespace)
	}

	err = client.Del(ctx, skey).Err()
	if err != nil {
		return fmt.Errorf("del cache key: %v err: %s", key, err.Error())
	}

	return nil
}

func (m *Cache) keyToString(key interface{}) (string, error) {
	switch t := key.(type) {
	case string:
		return t, nil
	case int8:
		return fmt.Sprintf("%d", key), nil
	case int16:
		return fmt.Sprintf("%d", key), nil
	case int32:
		return fmt.Sprintf("%d", key), nil
	case int64:
		return fmt.Sprintf("%d", key), nil
	case uint8:
		return fmt.Sprintf("%d", key), nil
	case uint16:
		return fmt.Sprintf("%d", key), nil
	case uint32:
		return fmt.Sprintf("%d", key), nil
	case uint64:
		return fmt.Sprintf("%d", key), nil
	case int:
		return fmt.Sprintf("%d", key), nil
	default:
		return "", errors.New("key err: unsupported type")
	}
}

func (m *Cache) fixKey(key interface{}) (string, error) {
	fun := "Cache.fixKey -->"

	skey, err := m.keyToString(key)
	if err != nil {
		slog.Errorf(context.TODO(), "%s key: %v err:%s", fun, key, err)
		return "", err
	}

	if len(m.prefix) > 0 {
		return fmt.Sprintf("%s.%s", m.prefix, skey), nil
	}

	return skey, nil
}

func (m *Cache) getValueFromCache(ctx context.Context, key, value interface{}) error {
	fun := "Cache.getValueFromCache -->"

	skey, err := m.fixKey(key)
	if err != nil {
		return err
	}

	client := redis.DefaultInstanceManager.GetInstance(ctx, m.namespace)
	if client == nil {
		slog.Errorf(ctx, "%s get instance err, namespace: %s", fun, m.namespace)
		return fmt.Errorf("get instance err, namespace: %s", m.namespace)
	}

	data, err := client.Get(ctx, skey).Bytes()
	if err != nil {
		return err
	}

	slog.Infof(ctx, "%s key: %v data: %s", fun, key, string(data))

	err = json.Unmarshal(data, value)
	if err != nil {
		return err
	}

	return nil
}

func (m *Cache) loadValueToCache(ctx context.Context, key interface{}) error {
	fun := "Cache.loadValueToCache -->"

	var data []byte
	value, err := m.load(key)
	if err != nil {
		slog.Warnf(ctx, "%s load err, cache key:%v err:%v", fun, key, err)
		data = []byte(err.Error())

	} else {
		data, err = json.Marshal(value)
		if err != nil {
			slog.Errorf(ctx, "%s marshal err, cache key:%v err:%v", fun, key, err)
			data = []byte(err.Error())
		}
	}

	skey, err := m.fixKey(key)
	if err != nil {
		slog.Errorf(ctx, "%s fixkey, key: %v err:%v", fun, key, err)
		return err
	}

	client := redis.DefaultInstanceManager.GetInstance(ctx, m.namespace)
	if client == nil {
		slog.Errorf(ctx, "%s get instance err, namespace: %s", fun, m.namespace)
		return fmt.Errorf("get instance err, namespace: %s", m.namespace)
	}

	rerr := client.Set(ctx, skey, data, m.expire).Err()
	if rerr != nil {
		slog.Errorf(ctx, "%s set err, cache key:%v rerr:%v", fun, key, rerr)
	}

	if err != nil {
		return err
	}

	return rerr
}

func SetConfiger(ctx context.Context, configerType cache.ConfigerType) error {
	fun := "Cache.SetConfiger-->"
	configer, err := redis.NewConfiger(configerType)
	if err != nil {
		slog.Errorf(ctx, "%s create configer err:%v", fun, err)
		return err
	}
	slog.Infof(ctx, "%s %v configer created", fun, configerType)
	redis.DefaultConfiger = configer
	return redis.DefaultConfiger.Init(ctx)
}

func init() {
	_ = SetConfiger(context.Background(), cache.ConfigerTypeSimple)
}
