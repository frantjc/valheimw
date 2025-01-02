package cache

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	xslice "github.com/frantjc/x/slice"
	kv "github.com/philippgille/gokv"
	"github.com/philippgille/gokv/encoding"
	"github.com/philippgille/gokv/gomap"
	"github.com/philippgille/gokv/redis"
	"github.com/philippgille/gokv/syncmap"
)

type Store = kv.Store

func NewStore(s string) (Store, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	codec := encoding.JSON

	switch u.Scheme {
	case "redis":
		userPassword, _ := u.User.Password()
		db, _ := strconv.Atoi(u.Query().Get("db"))
		var timeout *time.Duration = nil
		if duration, err := time.ParseDuration(u.Query().Get("timeout")); err == nil {
			timeout = &duration
		}

		return redis.NewClient(redis.Options{
			Address:  u.Host,
			Password: xslice.Coalesce(userPassword, os.Getenv("REDIS_PASSWORD")),
			DB:       db,
			Timeout:  timeout,
			Codec:    codec,
		})
	case "map", "gomap":
		return gomap.NewStore(gomap.Options{
			Codec: codec,
		}), nil
	case "mem", "", "syncmap":
		return syncmap.NewStore(syncmap.Options{
			Codec: codec,
		}), nil
	}

	return nil, fmt.Errorf("unknown scheme: %q", u.Scheme)
}
