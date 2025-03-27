# Simple Distributed Cache learning from scratch
- 学习分布式缓存的设计思路
- 学习[groupcache](https://github.com/golang/groupcache), 保留其核心功能
- 二次开发
  - 为lru`添加了TTL功能`
  - 实现了基于`rpc`的节点通信

## 技术点
1. `LRU`缓存淘汰策略, 添加了TTL过期
2. `一致性哈希算法`解决缓存服务器扩展或故障时缓存重建的问题
3. `singleflight`缓解大量访问热点数据造成的缓存击穿
4. 多种节点通信方式
   - http
   - rpc

## 架构
- ![poccess](/doc/poccess.png)
- ![topology](/doc/topology.png)

## 项目结构
```text
|   byteview.go // 只读缓存数据拷贝, 防止缓存值被外部程序修改
|   cache.go // 缓存
|   group.go // 缓存的命名空间
|   http.go // HTTP节点间通信服务器
|   peer.go // 远程节点抽象
|
+---consistenthash
|       consistenthash.go // 一致性哈希算法实现
|
+---lru
|       lru.go // LRU淘汰算法实现
|
\---singleflight
        singleflight.go // singleflight合并冗余请求, 防止因热点数据大量访问导致的缓存击穿
```

## Examples
- 构建了一个groupcache db的experiment
### 启动db服务器
```shell
go run /examples/dbserver/dbserver.go
```
### 启动Frontends
```shell
cd /examples/frontend
go run ./frontend -port 8001
go run ./frontend -port 8002
go run ./frontend -port 8003
```
### 启动cli进行数据操作
```shell
cd /exapmles/cli
go run ./cli -set -key foo -value bar
go run ./cli -cget -key foo # 第一次获取
go run ./cli -cget -key foo # 再次get, 从缓存中读取
```

## 学习笔记
### [学习记录](https://soongao.github.io/posts/cache/)
- 参考geecache