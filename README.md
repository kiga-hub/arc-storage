# Arc Storage 数据管理服务简述

数据管理服务是基于`Echo Web Framework`开发的后台数据管理服务，集成数据缓存(filecahe)、数据存储(文件、关系型数据库、时序性数据库)
数据接收(`kafka`,`gRPC`)。提供可扩展的`RESTful API`，对已存储的历史数据进行访问等服务。具有管理平台下所有数据的生命周期管理的功能。

## 主要功能

- 解析`kafka`缓存数据或`gRPC`传输的数据。
- 保存原始数据到arc文件。保存文件由采集设备ID、保存时间进行分类。
- 提供可扩展`RESTful API`，对数据进行访问和下载。
- 定期进行数据的拷贝和备份，管理生产环境设备数据的生命周期。


## 技术选型

- 后端:`Echo Web`框架构建的`RESTful API`。
- 消息中间件:使用`kafka`作为消息中间件，可以保证数据的一致性和可靠性，`kafka`是一个高性能、分布式的消息系统，广泛用于日志收集、流式数据处理、在线和离线消息分发等场景。
- 数据传输服务:使用`gRPC`服务，接收arc-comsumer发送的数据。使用`protobuf`来定义接口，提供更加严格的接口约束条件，通过`protobuf`可以将数据序列化为二进制编码，减少传输的数量。
- 数据库:采用`TDEngine`大数据平台(version: 2.0.14.0)，使用`taosSql`实现数据的存储操作。`TDEngine`专为物联网、车联网、工业互联网、IT运维等设计和优化的大数据平台。除核心的快10倍以上的时序数据库功能外，还提供缓存、数据订阅、流式计算等功能，最大程度减少研发和运维的复杂度.
- API文档:使用`Swagger`构建自动化文档。
- 配置:使用arc-storage.toml格式的配置文件。
- 日志:使用`zap-graylog`实现日志记录(loki)。

## 架构概览

### 目录结构

```bash
    ├─cmd                   (CLI命令行工具)
    ├─swagger               (本地swagger API文档依赖静态文件)
    ├─pkg                   (依赖包)
        ├─arc_grpc          (grpc数据转发)
        ├─cache             (实时数据缓存)
        ├─component         (组件注册)
        ├─arc_volume        (缓存落盘数据)
        ├─protostream       (数据序列化)
        |─kafka             (kafka数据接收)
        ├─util              (工具包)
        ├─handles.go        (初始化,数据包处理)
        |─storage.go        (历史文件查询接口实现)
        ...

```

### 数据管理服务功能文档

## 安装

### 源码编译(单机)

```bash
git clone github.com/kiga-hub/arc-storage.git
cd arc-storage
go mod tidy
go build
```

### 修改配置文件
```bash
vim arc-storage.toml
```

cahe 配置
```
search= true  查找时才开始缓存，false一直存储timeoutmin 失效
```

### 运行
```bash
# 拷贝arc-storage.toml 到可执行程序目录下
./arc-storage run
```

### swagger配置

移除`go-micro`依赖后的编译版本，执行程序调用`RESTful API`时会报以下错误

```bash
WARN	echozap@v1.1.1/logger.go:48	Client error	{"name": "arc-storage", "uuid": "000000ff-0000-1234-0000-0000000000ff", "v": "", "error": "code=404, message=Not Found", "remote_ip": "192.168.9.4", "time": "173.622µs", "host": "ip:8999 "request": "GET /api/data/v1/history/static/swagger/swagger-ui.css", "status": 404, "size": 24, "user_agent": "Mozilla/5.0 (Windows NT 10.0; Wi4; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36 Edg/92.0.902.67", "request_id": ""}
```

解决方法:
- 拷贝当前项目路径下swagger文件夹，到可执行程序目录。

### Docker镜像

```bash
#工作站拉取arc-storage镜像  image:dev
docker pull arc-storage:dev-linux-amd64

#导出镜像，并保存到指定位置
docker save -o ./arc-storage.tar arc-storage:dev-linux-amd64

#内网拷贝
#scp ./arc-storage.tar root@ip:/ 

#部署环境加载镜像
docker load -i ./arc-storage.tar

#镜像重新打tag并推送镜像到目标机registry
docker tag arc-storage:dev-linux-amd64 ip/platform/arc-storage:dev-linux-amd64
docker push ip/platform/arc-storage:dev-linux-amd64
```

### 命令行执行

```bash
# ./build.sh arc-storage arc-storage
./arc-storage run
```