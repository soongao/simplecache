package lru

import (
	"container/list"
	"time"
)

// Cache is a LRU cache. It is not safe for concurrent access.
// map + 双向queue, 规定队列Front为最近访问的元素, Back为最近最久未被访问元素
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value) // 删除时触发的callback, 可以为nil

	// 增加TTL
	Now NowFunc
}

type NowFunc func() time.Time

type entry struct {
	key   string
	value Value
	// TTL
	expire time.Time
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
		Now:       time.Now,
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (Value, bool) {
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry) // ele.Value是list.Element.Value type -> any type, 转换成*entry type
		if !kv.expire.IsZero() && kv.expire.Before(c.Now()) {
			c.RemoveElement(ele)
			return nil, false
		}
		c.ll.MoveToFront(ele) // 双向list, 队头和队尾是相对的, 这里规定Front是队尾
		return kv.value, true
	}
	return nil, false
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // Back是队头, 也就是要淘汰的元素
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value, expire time.Time) {
	if ele, ok := c.cache[key]; ok { // 如果key已存在, 修改元素
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
		kv.expire = expire

	} else { // key不存在, 插入元素
		ele := c.ll.PushFront(&entry{key, value, expire})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 如果插入后超出maxBytes了, 执行替换策略
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) RemoveElement(ele *list.Element) {
	c.ll.Remove(ele)
	kv := ele.Value.(*entry)
	delete(c.cache, kv.key)
	c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}
