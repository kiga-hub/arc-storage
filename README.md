# Arc Storage 数据管理服务简述

数据管理服务是基于`Echo Web Framework`开发的后台数据管理服务，集成数据缓存(filecahe)、数据存储(file,tao,HayStack)、
数据接收(`kafka`,`gRPC`)。提供可扩展的`RESTful API`，对已存储的历史数据进行访问等服务。具有管理平台下所有数据的生命周期管理的功能。

## 主要功能

- 解析`kafka`缓存数据或`gRPC`传输的数据。
- 保存音振原始数据到wav文件。保存文件由采集设备ID、保存时间进行分类。
- 提供可扩展`RESTful API`，对历史数据进行访问和下载。
- 定期进行数据的拷贝和备份，管理生产环境设备数据的生命周期。


## 技术选型

- 后端:`Echo Web`框架构建的`RESTful API`。
- 消息中间件:使用`kafka`作为消息中间件，可以保证数据的一致性和可靠性，`kafka`是一个高性能、分布式的消息系统，广泛用于日志收集、流式数据处理、在线和离线消息分发等场景。(长期测试后，从技术栈中移除，使用`gRPC`进行数据的传输)。
- 数据传输服务:使用`gRPC`服务，接收data-receiver发送的数据。使用`protobuf`来定义接口，提供更加严格的接口约束条件，通过`protobuf`可以将数据序列化为二进制编码，减少传输的数量。
- 数据库:采用`TDEngine`大数据平台(version: 2.0.14.0)，使用`taosSql`实现温度、振动及其他数据的存储操作。`TDEngine`专为物联网、车联网、工业互联网、IT运维等设计和优化的大数据平台。除核心的快10倍以上的时序数据库功能外，还提供缓存、数据订阅、流式计算等功能，最大程度减少研发和运维的复杂度.
- API文档:使用`Swagger`构建自动化文档。
- 配置:使用arc-storage.toml格式的配置文件。
- 日志:使用`zap-graylog`实现日志记录(loki)。
- ~~存储:使用小文件存储系统`haystack`，保存原始数据为一个大文件。~~

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
        ├─filecache         (缓存落盘数据)
        ├─protostream       (数据序列化)
        ├─storage           (haystack小文件存储)
        ├─tdengine          (数据库存储)
        ├─util              (工具包)
        ├─api.go            (接口)
        ├─decode.go         (数据解析)
        ├─handels_test.go   (测试文件)
        ├─handles.go        (初始化,数据包处理)
        ├─haystack.go       (haystack文件查询)
        |─kafka.go          (kafka数据接收)
        |─storage.go        (历史文件查询接口实现)
        |─taos.go           (数据库查询值数据)
```

### 数据管理服务功能文档

点击查看

>[数据管理服务功能文档](docs/数据管理服务功能文档.md)

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
   - 音频
   - 振动
   - 温度

2. 存储路径
    - 每个传感器ID对应一个目录,ID为`A00000000000`时,文件夹为ID对应字符串
      - `A00000000000`
    - 子目录包含以(年月日)为名的文件夹，按天为单位进行存储。音频保存至/audio/下，振动保存至/vibrate/下
      - `/A00000000000/20220330/audio/`
      - `/A00000000000/20220330/vibrate/`
    - 每分钟对缓存数据进行数据追加、新建文件等落盘操作

3. 存储格式
    - 缓存音振实时数据(cache)
    - 音频振动存储为wav文件
    - 温度解析为值数据之后存到TDEngine
    - 振动解析后存储三轴数据到TDEngine


4.  文件命名
    - 音频文件名格式：采集设备ID、音振类型、开始存储时间、结束时按、数据接收状态、采样率、固件、版本号 + arc-storage版本号
      - `A00000000000_A_20210317071415731_20210317071500019_N_48000_010000_01_000080_v1.0.95.wav`
      - 采集设备ID=A00000000000
      - 类型=A(音频-A  振动-V)
      - 开始时间=20210317071415731 = 2021-03-17 07:14:15.731
      - 结束时间=20210317071500019 = 2021-03-17 07:15:00.019
      - N=数据接收正常
      - 采样率=48000
      - 固件版本=010000
      - 硬件版本=01
      - 标志位=000000
      - arc-storage版本号=v1.0.95
    - 存储文件尾wav，包含44字节头文件

5. 音振数据属性
    - 音频48k相差20微秒；
    - 音频32k相差31微秒；
    - 音频16k相差62微秒；
    - 音频8k相差125微秒；

6. 文件压缩
    1. 设置ARC_SAVETYPE = 0 ，默认不压缩，直接输出为wav文件
    2. 压缩模式下使用BestSpeed，压缩率89.7~92.3%，平均每个文件压缩耗时39.7699ms~56.772487ms
    3. 压缩时，文件名后缀为wav.gz
    4. 压缩文件包含head数据,解压后为wav文件


### 数据切分-数据解包，对齐（只对filecache部分进行处理）

>数据缓存，写入，切分逻辑部分:

|       | isInterVal\SampleRate | isStartMin | isStopMin | ------- | Write |  Cut  | NewFile | Write |
| :---: | :-------------------: | :--------: | :-------: | :-----: | :---: | :---: | :-----: | :---: |
| Case1 |           ×           |     ×      |     ×     | ------- |   ×   |   ×   |    ×    |   √   |
| Case2 |           √           |            |           | ------- |   ×   |   √   |    √    |   √   |
| Case3 |           ×           |     ×      |     √     | ------- |   √   |   √   |    √    |   √   |
| Case4 |           ×           |     √      |     ×     | ------- |   ×   |   √   |    √    |   √   |

1. 写入Buffer
   1. 采样率不变
   2. 包号连续
   3. 当前Frame开始时间与结束时间在同一分钟时间段内

2. Buffer数据写入文件，开始重新缓存新数据
   1. 采样率改变
   2. 包号不连续
      - isInterrupt flag
   3. 当前Frame开始时间与结束时间不在同一分钟时间段内
   4. 服务异常退出，退出前Buffer数据写入文件
      - (写入文件，结束时间通过采样率重新计算结束时间)
   5. 接收数据，Frame时间戳超时，Buffer数据写入文件

3. 数据切分（整分钟写入wav文件时进行切分）
   1. 数据切分计算数据段:
        ```golang
        // segment1: Frame切分前段
        // segment2: Frame切分后段
        // 结束时间与零点进行比较，计算切分段大小
        segment2 = audioDatalen * int(afi.SaveTime.UTC().Sub(frameStopTime.UTC())) * int(afi.SampleRate) / 1e12
        segment1 = audioDatalen - segment2
        if segment2 < 0 || segment2 > audioDatalen {
            segment2 = 0
        }
        segment1 = segment1 - segment1%2
        segment2 = segment2 - segment2%2
        ```

   2. 切分场景1:   
        1. <img src="docs/cutframe_case1.png" width="60%">
        2. 根据当前数据采样率计算每分钟保存的采样点数
        3. 与缓存数据大小进行比较，当前混存数据(A) =当前帧(CurrentData) + 下一分钟采样点缓存数据(B)= 1分总采样点总数 
        4. 已经满足保存条件，则保存当前帧数据至下一分钟缓存中
            ```golang
            # 音频采样点数计算
            audioSize := int(segmentaudio.SampleRate) * 60 * 2
            # 振动采样点数计算
            vibrateSize := cast.ToInt(filecache.SamplerateConversion2Float64(segmentvibrate.SampleRate) * 60 * 6)
            ```

   3. 切分场景2:
        1. <img src="docs/cutframe_case2.png" width="60%">
        2. 当前缓存数据不足以进行落盘操作，需要切分数据帧
        3. 缓存数据(A) + index1 =index2 + 下一分钟采样点缓存数据(B) = 1分钟采样点总数 
        4. 满足保存条件，切分当前帧

   4. 切分场景3:
        1. <img src="docs/cutframe_case3.png" width="60%">
        2. 当前缓存数据(A) + 当前帧数据(CurrentData) = 下一分钟采样点缓存数据(B) = 1分钟采样点总数
        3. 满足保存条件，不进行切分，开始下一步操作 

4. 时间戳计算
    1. 计算结束时间点
        ```golang
        duration := float64(Buffer.Len()) / float64(SampleRate) / (2 * float64(Channel))
        ms, _ := time.ParseDuration("+" + cast.ToString(duration) + "s")
        saveTime := createTime.UTC().Add(ms)
        ```