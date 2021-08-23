package squirrel

import (
	"sync"
)

type Cache struct {
	stashes map[interface{}]*Stash
	lock sync.RWMutex

	find func(key interface{}) interface{}
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) UpsertValue(key interface{}, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.stashes[key] = NewStash(val).Now()
}

func (c *Cache) UpsertStash(key interface{}, s *Stash) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.stashes[key] = s
}

func (c *Cache) Get(key interface{}) interface{} {
	// Don't use a defer here, since c.UpsertValue might need the lock before the end of the func
	c.lock.RLock()

	// Try to find it in direct cache
	stash, found := c.stashes[key]
	if found {
		c.lock.RUnlock()
		return stash.val
	}

	c.lock.RUnlock()

	// Not in the cache. If find is set we use it to search for the value
	if c.find != nil {
		res := c.find(key)
		if res != nil {
			c.UpsertValue(key, res)
			return res
		}
	}

	// Not found anywhere
	return nil
}

func (c *Cache) UpdateIfNewer(key interface{}, s *Stash) {
	// Don't use a defer here, since c.UpsertStash might need the lock before the end of the func
	c.lock.RLock()

	// We avoid the Get function since we don't want to fall back to the c.find call
	current, found := c.stashes[key]

	c.lock.RUnlock()

	if !found {
		// No previous value. Just add it
		c.UpsertStash(key, s)
	}

	if s.status.creation.After(current.status.creation) {
		// The new value is newer. Upsert it
		c.UpsertStash(key, s)
	}

	// The new value is newer. Keep it
}

func (c *Cache) Delete(key interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.stashes, key)
}