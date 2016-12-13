package auth

import (
	"time"

	"github.com/pmylund/go-cache"
	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

// CacheAuthorization wraps an Authorizer in an in memory cache that expires
// after the given expiration. Only positive authorizations will be cached.
func CacheAuthorization(a Authorizer, expiration time.Duration) Authorizer {
	cache := cache.New(expiration, 30*time.Second)

	return &cachedAuthorizer{
		Authorizer: a,
		cache:      cache,
	}
}

// cachedAuthorizer is an Authorizer middleware that caches positive
// authorizations.
type cachedAuthorizer struct {
	Authorizer

	cache interface {
		Set(k string, x interface{}, d time.Duration)
		Get(k string) (interface{}, bool)
	}
}

func (a *cachedAuthorizer) Authorize(ctx context.Context, user *empire.User) error {
	_, ok := a.cache.Get(user.Name)

	// Authorized!
	if ok {
		return nil
	}

	// Not in cache, call down to the wrapped Authorizer.
	err := a.Authorizer.Authorize(ctx, user)

	// Only cache positive authorizations.
	if err == nil {
		a.cache.Set(user.Name, true, 0)
	}

	return err
}
