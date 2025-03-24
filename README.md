# Simple Distributed Cache from scratch
- 学习分布式缓存的设计思路
- 简单版的[groupcache](https://github.com/golang/groupcache), 保留其核心功能

## 项目结构
```text
|   byteview.go
|   cache.go
|   dcache.go
|   http.go
|   peer.go
|
+---cmd
|   +---api
|   |       api.go
|   |
|   \---peerserver
|           peerserver.go
|
+---consistenthash
|       consistenthash.go
|
+---lru
|       lru.go
|
+---pb
|       geecachepb.pb.go
|       geecachepb.proto
|
\---singleflight
        singleflight.go
```

## 使用Demo
```sh
go run ./cmd/api/api.go # 启动frontend api服务
# GET localhost:9999/api?key={key}
# key = Tom/Jack/Sam 用本地数据模拟数据库
go run ./cmd/peerserver/peerserver.go -port {port} # 启动远端
# port = 8001/8002/8003 模拟三个远端
```

## 学习笔记
### [学习记录](https://soongao.github.io/posts/cache/)
- 参考[geektutu](https://geektutu.com/post/geecache.html)