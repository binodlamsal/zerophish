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
var hits, misses, deletions uint64
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

	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for _ = range ticker.C {
			log.Infof("models.cache: items: %d, hits: %d, misses: %d, deletions: %d", cache.c.ItemCount(), hits, misses, deletions)
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

// AddRoleDisplayName adds a given role display name to the cache
func (cache *Cache) AddRoleDisplayName(rid int64, name string) {
	if disable {
		return
	}

	cache.AddEntry("role", rid, "display_name", name)
}

// AddUser adds a given user to the cache
func (cache *Cache) AddUser(u *User) {
	if disable {
		return
	}

	cache.AddEntry("user", u.Id, "", u)
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

// GetRoleDisplayName finds and returns cached display name of a role with the given rid
func (cache *Cache) GetRoleDisplayName(rid int64) (string, bool) {
	if disable {
		return "", false
	}

	key := fmt.Sprintf("role.%s.display_name", strconv.FormatInt(rid, 10))

	if val, found := cache.c.Get(key); found {
		if rname, ok := val.(string); ok {
			logCacheGet(key, true)
			return rname, true
		}

		logCacheGet(key, true)
		return "", true
	}

	logCacheGet(key, false)
	return "", false
}

// GetUserById finds and returns cached user with a given id
func (cache *Cache) GetUserById(uid int64) (*User, bool) {
	if disable {
		return nil, false
	}

	key := fmt.Sprintf("user.%s", strconv.FormatInt(uid, 10))

	if val, found := cache.c.Get(key); found {
		if u, ok := val.(*User); ok {
			logCacheGet(key, true)
			return u, true
		}

		logCacheGet(key, true)
		return nil, true
	}

	logCacheGet(key, false)
	return nil, false
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

// DeleteUser removes a given user from the cache
func (cache *Cache) DeleteUser(u *User) {
	if disable {
		return
	}

	key := fmt.Sprintf("user.%s", u.Id)
	cache.c.Delete(key)
	logCacheDel(key)
}

func logCacheAdd(key string) {
	if logOps {
		log.Infof("models.cache: add %s", key)
	}
}

func logCacheGet(key string, hit bool) {
	if hit {
		hits++
	} else {
		misses++
	}

	if logOps && !hit {
		log.Infof("models.cache: miss %s", key)
	}
}

func logCacheDel(key string) {
	deletions++

	if logOps {
		log.Infof("models.cache: delete %s", key)
	}
}
