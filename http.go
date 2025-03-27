package dcache

import (
	"dcache/consistenthash"
	"dcache/pb"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	defaultBasePath = "/_dcache_/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string     // addr host + port
	basePath    string     // url prefix /<basepath>/<groupname>/<key>
	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	expire := r.URL.Query().Get("expire")
	var expireTime time.Time
	if expire == "" {
		expireTime = time.Time(time.Unix(0, 0))
	} else {
		ns, err := strconv.Atoi(expire)
		if err != nil {
			http.Error(w, "expire wrong type "+expire, http.StatusNotFound)
			return
		}
		expireTime = time.Now()
		expireTime.Add(time.Second * time.Duration(ns))
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key, expireTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the value to the response body as a proto message.
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()}) // 传输数据使用protobuf进行压缩
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

type httpGetter struct {
	baseURL string
}

var _ PeerGetter = (*httpGetter)(nil) // 类型转换, 确保*httpGetter实现了PeerGetter接口, 保证健壮性

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil { // 对protobuf压缩的数据解码
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

// Set updates the pool's list of peers.
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

var _ PeerPicker = (*HTTPPool)(nil)

// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
// func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	if !strings.HasPrefix(r.URL.Path, p.basePath) {
// 		panic("HTTPPool serving unexpected path: " + r.URL.Path)
// 	}
// 	p.Log("%s %s", r.Method, r.URL.Path)
// 	// /<basepath>/<groupname>/<key> required
// 	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
// 	if len(parts) != 2 {
// 		http.Error(w, "bad request", http.StatusBadRequest)
// 		return
// 	}

// 	groupName := parts[0]
// 	key := parts[1]

// 	group := GetGroup(groupName)
// 	if group == nil {
// 		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
// 		return
// 	}

// 	view, err := group.Get(key)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/octet-stream")
// 	w.Write(view.ByteSlice())
// }

// 向peer发送http请求
// func (h *httpGetter) Get(group string, key string) ([]byte, error) {
// 	u := fmt.Sprintf(
// 		"%v%v/%v",
// 		h.baseURL,
// 		url.QueryEscape(group),
// 		url.QueryEscape(key),
// 	) // <baseURL = addr+basePath>/<groupname>/<key>
// 	res, err := http.Get(u) // 向远端peer发送Get请求
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer res.Body.Close()

// 	if res.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("server returned: %v", res.Status)
// 	}

// 	bytes, err := io.ReadAll(res.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("reading response body: %v", err)
// 	}

//		return bytes, nil
//	}
