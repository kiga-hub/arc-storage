# Arc Storage 数据管理服务简述

数据管理服务是基于`Echo Web Framework`开发的后台数据管理服务，集成数据缓存(filecahe)、数据存储(file,tao,HayStack)、
数据接收(`kafka`,`gRPC`)。提供可扩展的`RESTful API`，对已存储的历史数据进行访问等服务。具有管理平台下所有数据的生命周期管理的功能。

## 主要功能

- 解析`kafka`缓存数据或`gRPC`传输的数据。
- 保存原始数据到wav文件。保存文件由采集设备ID、保存时间进行分类。
- 提供可扩展`RESTful API`，对历史数据进行访问和下载。
- 定期进行数据的拷贝和备份，管理生产环境设备数据的生命周期。


## 技术选型

- 后端:`Echo Web`框架构建的`RESTful API`。
- 消息中间件:使用`kafka`作为消息中间件，可以保证数据的一致性和可靠性，`kafka`是一个高性能、分布式的消息系统，广泛用于日志收集、流式数据处理、在线和离线消息分发等场景。(长期测试后，从技术栈中移除，使用`gRPC`进行数据的传输)。
- 数据传输服务:使用`gRPC`服务，接收data-receiver发送的数据。使用`protobuf`来定义接口，提供更加严格的接口约束条件，通过`protobuf`可以将数据序列化为二进制编码，减少传输的数量。
- 数据库:采用`TDEngine`大数据平台(version: 2.0.14.0)，使用`taosSql`实现数据的存储操作。`TDEngine`专为物联网、车联网、工业互联网、IT运维等设计和优化的大数据平台。除核心的快10倍以上的时序数据库功能外，还提供缓存、数据订阅、流式计算等功能，最大程度减少研发和运维的复杂度.
- API文档:使用`Swagger`构建自动化文档。
- 配置:使用arc-storage.toml格式的配置文件。
- 日志:使用`zap-graylog`实现日志记录(loki)。

## 架构概览

### 目录结构

```bash
    ├─api                   (SDK API)
    ├─cmd                   (CLI命令行工具)
    ├─docs                  (文档)
    ├─swagger               (本地swagger API文档依赖静态文件)
    ├─pkg                   (依赖包)
        ├─arc_grpc          (grpc数据转发)
        ├─api               (数据相关结构体 TODO修改格式&命名)
        ├─cache             (实时数据缓存)
        ├─component         (组件注册)
        ├─crontab           (数据迁移定时任务)
        ├─arc_volume        (缓存落盘数据)
        ├─protostream       (数据序列化)
        ├─storage           (haystack小文件存储)
        ├─tdengine          (数据库存储)
        |─kafka             (kafka数据接收)
        ├─util              (工具包)
        ├─api.go            (接口)
        ├─decode.go         (数据解析)
        ├─handels_test.go   (测试文件)
        ├─handles.go        (初始化,数据包处理)
        |─storage.go        (历史文件查询接口实现)
        |─taos.go           (数据库查询值数据)
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
WARN	echozap@v1.1.1/logger.go:48	Client error	{"name": "arc-storage", "uuid": "000000ff-0000-1234-0000-0000000000ff", "v": "", "error": "code=404, message=Not Found", "remote_ip": "192.168.9.45", "time": "173.622µs", "host": "192.168.8.245:8999 "request": "GET /api/data/v1/history/static/swagger/swagger-ui.css", "status": 404, "size": 24, "user_agent": "Mozilla/5.0 (Windows NT 10.0; Wi4; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36 Edg/92.0.902.67", "request_id": ""}
```

解决方法:
- 拷贝当前项目路径下swagger文件夹，到可执行程序目录。

### Docker镜像

```bash
#工作站拉取arc-storage镜像  image:12345678
docker pull arc-storage:12345678-linux-amd64

#导出镜像，并保存到指定位置
docker save -o ./arc-storage.tar arc-storage:12345678-linux-amd64

#内网拷贝
#scp ./arc-storage.tar root@192.168.8.253:/ 

#部署环境加载镜像
docker load -i ./arc-storage.tar

#镜像重新打tag并推送镜像到目标机registry
docker tag arc-storage:12345678-linux-amd64 192.168.8.253/platform/arc-storage:12345678-linux-amd64
docker push 192.168.8.253/platform/arc-storage:12345678-linux-amd64
```

##  数据处理

- 使用`gRPC`接收数据并反序列化，由数据管理服务(arc-storage)进行下一步数据处理

### 数据包协议

```golang
type Frame struct {
	Head      [4]byte //4 头标志 0xFC 0xFC 0xFC 0xFC
	Version   byte    //1 包格式版本 = 1
	Size      uint32  //4 包大小 [Timestamp, End] = 32+n BigEndian
	Timestamp int64   //8 时间戳/序号 精确到毫秒 BigEndian
	BasicInfo [6]byte //6 (2-客户 1-设备 1-年份 1-月份 1-日期)
	ID        [6]byte //6 设备编号 (Mac Address)
	Firmware  [3]byte //3 固件版本3位
	Hardware  byte    //1 硬件版本1位
	Protocol  [2]byte //2 协议版本 = 1 BigEndian
	Flag      [3]byte //3 标志位 前8位表示数据形式 AVT_____ 第9位表示 有线/无线 其他预留
	DataGroup []byte  //n 数据
	Crc uint16        //2 校验位 [Timestamp, Data], CRC-16 BigEndian
	End [1]byte       //1 结束标志 0xFD
}
type DataGroup struct {
	Count    byte     //1 数据类型个数
	Sizes    []uint32 //4 每个类型数据大小
	Segments []byte   //n 数据
}
```

### 数据存储

1. 存储数据类型
   - 模拟数据Arc


3. 存储格式
    - 

1. 写入Buffer
   1. 包号连续
   2. 当前Frame开始时间与结束时间在同一分钟时间段内

2. Buffer数据写入文件，开始重新缓存新数据
   1. 包号不连续
      - isInterrupt flag
   3. 当前Frame开始时间与结束时间不在同一分钟时间段内
   4. 服务异常退出，退出前Buffer数据写入文件
      - (写入文件，结束时间通过采样率重新计算结束时间)
   5. 接收数据，Frame时间戳超时，Buffer数据写入文件

4. 时间戳计算
    1. 计算结束时间点
        ```golang
        duration := float64(Buffer.Len()) / float64(SampleRate) / (2 * float64(Channel))
        ms, _ := time.ParseDuration("+" + cast.ToString(duration) + "s")
        saveTime := createTime.UTC().Add(ms)
        ```