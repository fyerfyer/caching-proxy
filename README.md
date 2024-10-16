> Project page URI: https://roadmap.sh/projects/caching-server

### 1.使用方法

&emsp;&emsp;使用如下语句运行go文件、初始化代理端口、转送请求的服务器地址和数据库地址：

```bash
go run . --port YourPort --origin YourOrigin -dsn: YourDSN 
```

&emsp;&emsp;使用curl对代理缓存的功能性进行测试：

#### 1.代理服务器

* **运行代理服务器**

```bash
./caching-proxy --port 3000 --origin http://dummyjson.com
```

* **客户端发送请求**
```bash
curl -i http://localhost:3000/products
```

* **清空缓存**
```bash
curl -X POST http://localhost:3000/clear-cache
```

#### 2.缓存API

* **添加缓存项**

```bash
curl -X POST http://localhost:3000/api/cache/add -d '{"key": "test", "data": "some data", "ttl": 3600}'
```

* **删除缓存项**

```bash
curl -X DELETE http://localhost:3000/api/cache/delete?key=test
```

* **查询缓存项**

```bash
curl -X GET http://localhost:3000/api/cache/get?key=test
```

### 2.代理缓存设计概述

#### 1.文件结构

&emsp;&emsp;项目实现的文件结构如下：

```go
/blogging-api
  ├── main.go     // setting up cache & read client parameters & running server
  ├── models.go   // the cache table model
  ├── routes.go   // the routers creation
  ├── handlers.go // the handlers for each router
  ├── cache.go    // the cache implementation 
```

#### 2.代理实现

&emsp;&emsp;我们使用`httputil.NewSingleHostReverseProxy`创建一个反向代理：

```go
// handlers.go
proxy := httputil.NewSingleHostReverseProxy(toUrl)
```

&emsp;&emsp;通过`proxy.Director`自定义请求转发行为：

```go
// handlers.go
proxy.Director = func(req *http.Request) {
	req.Host = toUrl.Host
	req.URL.Scheme = toUrl.Scheme
	req.URL.Host = toUrl.Host
}
```

&emsp;&emsp;通过`proxy.ModifyResponse`修改目标服务器返回给客户端的响应，将响应的具体内容写入`http.ResponseWriter`中：

```go
proxy.ModifyResponse = func(r *http.Response) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	app.Cache.Set(cacheKey, string(body), 5*time.Minute)
	w.Header().Set("X-Cache", "MISS")
	w.Write(body)
	return nil
}
```

&emsp;&emsp;最后启动我们设置好的代理服务器：

```go
proxy.ServeHTTP(w, r)
```

#### 3.缓存实现

&emsp;&emsp;缓存的结构体定义如下：

```go
type CacheItem struct {
	Key        string        `json:"key"`
	Value      string        `json:"value"`
	TTL        time.Duration `json:"ttl"`
	Expiration time.Time     `json:"expiration"`
}
```

&emsp;&emsp;我们采用内存+数据库的双重存储方式来管理缓存，定义结构体如下：

```go
// models.go
type Cache struct {
	items map[string]*CacheItem
	mu    sync.Mutex
	DB    *sql.DB
}
```

* 首先在内存中寻找某缓存是否存在。
* 如果未找到，再在数据库中寻找。


&emsp;&emsp;在初始化缓存时，我们加入一个用于清理过期缓存的goroutine，并用`time.NewTicker`创建一个ticker来周期性地触发它：

```go
// cache.go
func (c *Cache) newCache(DB *sql.DB) *Cache {
	go c.startExpirationHandler(5 * time.Minute) // clean expired cache periodically 
	return &Cache{
		items: make(map[string]*CacheItem),
		DB:    DB,
	}
}
```