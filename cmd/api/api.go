package main

import (
	"dcache"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *dcache.Group {
	return dcache.NewGroup("scores", 2<<10, dcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startAPIServer(apiAddr string, d *dcache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := d.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	// singleflight.Group
	apiAddr := "http://localhost:9999"
	group := createGroup()
	// 假设缓存服务器地址
	peers := dcache.NewHTTPPool("http://localhost:9999")
	peers.Set("http://localhost:8001", "http://localhost:8002", "http://localhost:8003")
	group.RegisterPeers(peers)

	startAPIServer(apiAddr, group)
}
