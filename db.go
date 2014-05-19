package main

import (
	"bytes"
	// "encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"net"
	"net/http"
	"strconv"
	"time"
)

var redisPool *redis.Pool

/*
 * Public API
 */

// Initialize subsystem. Must preceed API calls to storage subsys.
func InitStorage() {
	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", net.JoinHostPort("127.1", "6379"))
			if err != nil {
				return nil, err
				// TODO при недоступности редиса останавливать мониторинг
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// Save monitoring data to DB.
func RedKeepResult(key Key, weight time.Time, res Result) error {
	//var buf bytes.Buffer

	conn := redisPool.Get()
	defer conn.Close()
	// enc := gob.NewEncoder(&buf)
	keepit := KeepedResult{
		Tid: res.Task.Tid,
		Stream: Stream{
			URI:   res.Task.URI,
			Type:  res.Task.Type,
			Name:  res.Task.Name,
			Group: res.Task.Group,
			Title: res.Task.Title,
		},
		ErrType:           res.ErrType,
		HTTPCode:          res.HTTPCode,
		HTTPStatus:        res.HTTPStatus,
		ContentLength:     res.ContentLength,
		RealContentLength: res.RealContentLength,
		Headers:           res.Headers,
		Body:              res.Body.Bytes(),
		Started:           res.Started,
		Elapsed:           res.Elapsed,
		TotalErrs:         res.TotalErrs,
	}
	if res.Pid == nil {
		keepit.Master = true
	} else {
		keepit.Master = false
	}
	buf, err := json.Marshal(keepit)
	// err := enc.Encode(&keepit)
	if err != nil {
		fmt.Printf("redis: %s\n", err)
		return err
	}
	_, err = conn.Do("ZADD", key.String(), strconv.FormatInt(weight.Unix(), 10), buf)
	if err != nil {
		fmt.Printf("redis RedKeepResult: %s\n", err)
	}
	return err
}

// Keeps values only for errors and warngings.
func RedKeepError(key Key, weight time.Time, errtype ErrType) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("ZADD", fmt.Sprintf("errors/%s", key.String()), strconv.FormatInt(weight.Unix(), 10), errtype)
	if err != nil {
		fmt.Printf("redis RedKeepError: %s\n", err)
	}
	return err
}

func RedLoadResults(key Key, from, to time.Time) ([]KeepedResult, error) {
	var src bytes.Buffer
	var dst KeepedResult = KeepedResult{Stream: Stream{}, Headers: make(http.Header)}
	var result []KeepedResult

	conn := redisPool.Get()
	defer conn.Close()
	data, err := redis.Values(conn.Do("ZRANGEBYSCORE", key.String(), strconv.FormatInt(from.Unix(), 10), strconv.FormatInt(to.Unix(), 10)))
	// dec := gob.NewDecoder(&src)
	for _, val := range data {
		src.Write(val.([]byte))
		//err := dec.Decode(&dst)
		err := json.Unmarshal(src.Bytes(), &dst)
		if err == nil {
			result = append(result, dst)
		}
		src.Reset()
	}
	return result, err
}

func RedLoadErrors(key Key, from, to time.Time) (map[time.Time]ErrType, error) {
	var retval ErrType

	conn := redisPool.Get()
	defer conn.Close()
	result := make(map[time.Time]ErrType)
	data, err := redis.Values(conn.Do("ZRANGEBYSCORE", fmt.Sprintf("errors/%s", key.String()), strconv.FormatInt(from.Unix(), 10), strconv.FormatInt(to.Unix(), 10), "WITHSCORES"))
	if err == nil {
		for idx, val := range data {
			if idx%2 == 0 { // data
				retval = ErrType(val.([]uint8)[0])
			} else { // key
				key, err := strconv.ParseInt(string(val.([]byte)), 10, 64)
				if err == nil {
					result[time.Unix(key, 0)] = retval
				}
			}
		}
		return result, nil
	} else {
		return nil, err
	}
}

// Remove expired errors from the Redis sorted set
func RemoveExpiredErrors(expired time.Duration) {
	conn := redisPool.Get()
	defer conn.Close()
	for gname := range cfg.GroupParams {
		for _, stream := range *cfg.GroupStreams[gname] {
			key := Key{gname, stream.Name}
			if deleted, _ := redis.Int(conn.Do("ZREMRANGEBYSCORE", fmt.Sprintf("errors/%s", key.String()), "-inf", strconv.FormatInt(time.Now().Add(-expired).Unix(), 10))); deleted > 0 {
				// fmt.Printf("%d expired elements from `errors` set `%s` deleted\n", deleted, key)
			}
		}
	}
}

// Remove expired errors from the Redis sorted set
func RemoveExpiredResults(expired time.Duration) {
	conn := redisPool.Get()
	defer conn.Close()
	for gname := range cfg.GroupParams {
		for _, stream := range *cfg.GroupStreams[gname] {
			key := Key{gname, stream.Name}
			if deleted, _ := redis.Int(conn.Do("ZREMRANGEBYSCORE", fmt.Sprintf("%s", key.String()), "-inf", strconv.FormatInt(time.Now().Add(-expired).Unix(), 10))); deleted > 0 {
				// fmt.Printf("%d expired elements from `results` set `%s` deleted\n", deleted, key)
			}
		}
	}
}
