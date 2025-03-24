package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash           // 使用的hash func
	replicas int            // 虚节点个数s
	keys     []int          // Sorted, 哈希环
	hashMap  map[int]string // 虚节点与真实节点的映射
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
// 这里是添加节点, 也就是peer
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := range m.replicas { // 对每一个peer, 它们的name=key, 并创建若干个虚节点
			// 构造如peer1_1, peer1_2等节点, 它们name的hash放入哈希环中
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) // 这里转换成int只是为了sort等后续操作方便, 哈希环还是uint的
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
// 查找任意一个key所对应的peer节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool { // 当没找到比hash大的peer hash时, 返回的idx为len(m.keys), 但这里是环形结构(逻辑上), uint没找到更大的值(顺时针转了一圈)
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]] // 当idx == len(m.keys)时, 应该是第0个peer, 因此%操作
}
