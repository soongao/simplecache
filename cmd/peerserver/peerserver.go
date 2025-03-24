package main

import (
	"dcache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

// 模拟数据库
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建缓存组
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

// 启动缓存服务器
func startCacheServer(addr string) {
	// log.Println(addr)
	peers := dcache.NewHTTPPool(addr)
	log.Println("dcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func main() {
	var port int
	// 解析命令行参数
	flag.IntVar(&port, "port", 8001, "dcache server port")
	flag.Parse()

	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// var addrs []string
	// for _, v := range addrMap {
	// 	addrs = append(addrs, v)
	// }

	createGroup()
	startCacheServer(addrMap[port])
}
