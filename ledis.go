// Copyright 2013 Beego Authors
// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package ledistore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/tango-contrib/session"

	"github.com/lunny/log"
	"github.com/lunny/tango"
	"github.com/siddontang/goredis"
)

var _ session.Store = &LedisStore{}

// Options describes the options for create ledis store
type Options struct {
	Host    string
	Port    string
	DbIndex int
	MaxAge  time.Duration
	Logger  tango.Logger
}

// LedisStore represents a ledis session store implementation.
type LedisStore struct {
	Options
	client *goredis.Client
}

func preOptions(opts []Options) Options {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Host == "" {
		opt.Host = "127.0.0.1"
	}
	if opt.Port == "" {
		opt.Port = "6380"
	}
	if opt.MaxAge == 0 {
		opt.MaxAge = session.DefaultMaxAge
	}
	if opt.Logger == nil {
		opt.Logger = log.Std
	}
	return opt
}

// NewLedisStore creates and returns a ledis session store.
func New(opts ...Options) (*LedisStore, error) {
	opt := preOptions(opts)

	var ledis = LedisStore{
		Options: opt,
		client:  goredis.NewClient(opt.Host+":"+opt.Port, ""),
	}

	if _, err := ledis.Do("PING"); err != nil {
		return nil, err
	}

	_, err := ledis.Do("SELECT", opt.DbIndex)
	if err != nil {
		return nil, err
	}

	return &ledis, nil
}

func (c *LedisStore) serialize(value interface{}) ([]byte, error) {
	err := c.registerGobConcreteType(value)
	if err != nil {
		return nil, err
	}

	if reflect.TypeOf(value).Kind() == reflect.Struct {
		return nil, fmt.Errorf("serialize func only take pointer of a struct")
	}

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)

	err = encoder.Encode(&value)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c *LedisStore) deserialize(byt []byte) (ptr interface{}, err error) {
	b := bytes.NewBuffer(byt)
	decoder := gob.NewDecoder(b)

	var p interface{}
	err = decoder.Decode(&p)
	if err != nil {
		return
	}

	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Struct {
		var pp interface{} = &p
		datas := reflect.ValueOf(pp).Elem().InterfaceData()

		sp := reflect.NewAt(v.Type(),
			unsafe.Pointer(datas[1])).Interface()
		ptr = sp
	} else {
		ptr = p
	}
	return
}

func (c *LedisStore) registerGobConcreteType(value interface{}) error {
	t := reflect.TypeOf(value)

	switch t.Kind() {
	case reflect.Ptr:
		v := reflect.ValueOf(value)
		i := v.Elem().Interface()
		gob.Register(i)
	case reflect.Struct, reflect.Map, reflect.Slice:
		gob.Register(value)
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Bool, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		// do nothing since already registered known type
	default:
		return fmt.Errorf("unhandled type: %v", t)
	}
	return nil
}

// Do execute command
func (s *LedisStore) Do(cmd string, args ...interface{}) (interface{}, error) {
	conn, err := s.client.Get()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.Do(cmd, args...)
}

// Set sets value to given key in session.
func (s *LedisStore) Set(id session.Id, key string, val interface{}) error {
	bs, err := s.serialize(val)
	if err != nil {
		return err
	}
	_, err = s.Do("HSET", id, key, bs)
	if err == nil {
		// when write data, reset maxage
		_, err = s.Do("HEXPIRE", id, s.MaxAge)
	}
	return err
}

// Get gets value by given key in session.
func (s *LedisStore) Get(id session.Id, key string) interface{} {
	val, err := s.Do("HGET", id, key)
	if err != nil {
		s.Logger.Errorf("ledis HGET failed: %s", err)
		return nil
	}

	// when read data, reset maxage
	s.Do("HEXPIRE", id, s.MaxAge)

	item, err := goredis.Bytes(val, err)
	if err != nil {
		s.Logger.Errorf("redis.Bytes failed: %s", err)
		return nil
	}

	value, err := s.deserialize(item)
	if err != nil {
		s.Logger.Errorf("ledis HGET failed: %s", err)
		return nil
	}
	return value
}

// Delete delete a key from session.
func (s *LedisStore) Del(id session.Id, key string) bool {
	_, err := s.Do("HDEL", id, key)
	return err == nil
}

func (s *LedisStore) Clear(id session.Id) bool {
	_, err := s.Do("HCLEAR", id)
	return err == nil
}

func (s *LedisStore) Add(id session.Id) bool {
	return true
}

func (s *LedisStore) Exist(id session.Id) bool {
	b, err := s.Do("HLEN", id)
	v, _ := goredis.Int(b, err)
	return v > 0
}

func (s *LedisStore) SetMaxAge(maxAge time.Duration) {
	s.MaxAge = maxAge
}

func (s *LedisStore) SetIdMaxAge(id session.Id, maxAge time.Duration) {
	if s.Exist(id) {
		s.Do("HEXPIRE", id, s.MaxAge)
	}
}

func (s *LedisStore) Run() error {
	return nil
}
