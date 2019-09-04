package models

import (
	"fmt"
	"os"
	"strconv"
	"time"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
	gcache "github.com/patrickmn/go-cache"
)

var cache *Cache
var logOps bool
var disable bool
var ttl time.Duration

// Cache wrapper around go-cache
type Cache struct {
	c *gcache.Cache
}

func init() {
	ttl = gcache.NoExpiration

	if minutes, err := strconv.Atoi(os.Getenv("CACHE_TTL_MINUTES")); err == nil {
		ttl = time.Duration(minutes) * time.Minute
		log.Infof("models.cache: TTL set to %s", ttl)
	}

	cache = &Cache{gcache.New(ttl, ttl)}

	if os.Getenv("CACHE_DISABLE") != "" {
		disable = true
	}

	if os.Getenv("CACHE_LOG_OPS") != "" {
		logOps = true
	}

	cache.c.OnEvicted(func(key string, val interface{}) {
		if logOps {
			log.Infof("models.cache: auto-delete %s", key)
		}
	})

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for _ = range ticker.C {
			log.Infof("models.cache: items: %d", cache.c.ItemCount())
		}
	}()
}

// GetCache returns singleton instance of the model cache
func GetCache() *Cache {
	return cache
}

// AddEntry adds given value to the cache with the key built from prefix, id and suffix
func (cache *Cache) AddEntry(prefix string, id int64, suffix string, val interface{}) {
	if disable {
		return
	}

	key := fmt.Sprintf("%s.%s.%s", prefix, strconv.FormatInt(id, 10), suffix)
	cache.c.SetDefault(key, val)
	logCacheAdd(key)
}

// AddUserSubscription adds a given subscription to the cache
func (cache *Cache) AddUserSubscription(s *Subscription) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s.subscription", strconv.FormatInt(s.UserId, 10))
	cache.c.SetDefault(key, s)
	logCacheAdd(key)
}

// AddUserAvatar adds a given avatar to the cache
func (cache *Cache) AddUserAvatar(a *Avatar) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s.avatar", strconv.FormatInt(a.UserId, 10))
	cache.c.SetDefault(key, a)
	logCacheAdd(key)
}

// AddUserRole adds a given user role to the cache
func (cache *Cache) AddUserRole(r *UserRole) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s.role", strconv.FormatInt(r.Uid, 10))
	cache.c.SetDefault(key, r)
	logCacheAdd(key)
}

// GetUserSubscription finds and returns cached subscription by its user id
func (cache *Cache) GetUserSubscription(uid int64) (*Subscription, bool) {
	if disable {
		return nil, false
	}

	key := fmt.Sprintf("user.%s.subscription", strconv.FormatInt(uid, 10))

	if val, found := cache.c.Get(key); found {
		if s, ok := val.(*Subscription); ok {
			logCacheGet(key, true)
			return s, true
		}

		logCacheGet(key, true)
		return nil, true
	}

	logCacheGet(key, false)
	return nil, false
}

// GetUserAvatar finds and returns cached avatar by its user id
func (cache *Cache) GetUserAvatar(uid int64) (*Avatar, bool) {
	if disable {
		return nil, false
	}

	key := fmt.Sprintf("user.%s.avatar", strconv.FormatInt(uid, 10))

	if val, found := cache.c.Get(key); found {
		if a, ok := val.(*Avatar); ok {
			logCacheGet(key, true)
			return a, true
		}

		logCacheGet(key, true)
		return nil, true
	}

	logCacheGet(key, false)
	return nil, false
}

// GetUserRole finds and returns cached role by its user id
func (cache *Cache) GetUserRole(uid int64) (*UserRole, bool) {
	if disable {
		return nil, false
	}

	key := fmt.Sprintf("user.%s.role", strconv.FormatInt(uid, 10))

	if val, found := cache.c.Get(key); found {
		if r, ok := val.(*UserRole); ok {
			logCacheGet(key, true)
			return r, true
		}

		logCacheGet(key, true)
		return nil, true
	}

	logCacheGet(key, false)
	return nil, false
}

// DeleteEntry deletes an entry from the cache using the key built from prefix, id and suffix
func (cache *Cache) DeleteEntry(prefix string, id int64, suffix string) {
	if disable {
		return
	}

	key := fmt.Sprintf("%s.%s.%s", prefix, strconv.FormatInt(id, 10), suffix)
	cache.c.Delete(key)
	logCacheDel(key)
}

// DeleteUserSubscription removes a given subscription from the cache
func (cache *Cache) DeleteUserSubscription(s *Subscription) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s.subscription", strconv.FormatInt(s.UserId, 10))
	cache.c.Delete(key)
	logCacheDel(key)
}

// DeleteUserAvatar removes a given avatar from the cache
func (cache *Cache) DeleteUserAvatar(a *Avatar) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s.avatar", strconv.FormatInt(a.UserId, 10))
	cache.c.Delete(key)
	logCacheDel(key)
}

// DeleteUserRole removes a given role from the cache
func (cache *Cache) DeleteUserRole(r *UserRole) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s.role", strconv.FormatInt(r.Uid, 10))
	cache.c.Delete(key)
	logCacheDel(key)
}

func logCacheAdd(key string) {
	if logOps {
		log.Infof("models.cache: add %s", key)
	}
}

func logCacheGet(key string, hit bool) {
	if logOps && !hit {
		log.Infof("models.cache: miss %s", key)
	}
}

func logCacheDel(key string) {
	if logOps {
		log.Infof("models.cache: delete %s", key)
	}
}
