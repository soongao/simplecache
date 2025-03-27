package dcache

import (
	"dcache/pb"
	"dcache/singleflight"
	"fmt"
	"log"
	"sync"
	"time"
)

// A Getter loads data for a key.
// 当缓存未命中时, 从Getter中读取数据, 这个Getter可以是文件、数据库等
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get value for a key from cache
func (g *Group) Get(key string, expire time.Time) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[DCache] hit")
		return v, nil
	}

	// 缓存未命中
	return g.load(key, expire)
}

func (g *Group) load(key string, expire time.Time) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[DCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key, expire)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// 从Getter中Get(key)
func (g *Group) getLocally(key string, expire time.Time) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	// 添加到cache中
	g.populateCache(key, value, expire)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView, expire time.Time) {
	g.mainCache.add(key, value, expire)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

// func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
// 	bytes, err := peer.Get(g.name, key)
// 	if err != nil {
// 		return ByteView{}, err
// 	}
// 	return ByteView{b: bytes}, nil
// }
