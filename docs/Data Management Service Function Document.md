## 数据管理服务功能概述

arc-storage数据管理服务通过arc-consmuer服务的数据中转，实现实时数据的落盘。

数据管理服务主要功能包括原始数据和解析数据的存储、部分数据的生命周期管理、数据定时备份、拷贝，历史数据的查询。

整个数据管理架构参考 [架构](../README.md)

### 功能划分

数据管理服务主要包含以下三个功能模块
- 历史数据查询
- 采集数据存储
- 数据生命周期管理

### 功能描述

1. 平台的所有服务使用docker容器化部署，各个服务通过指定路由进行内网的通讯
2. 数据接收服务收到硬件发上来的数据后，进行解析和对数据包时间的处理。之后将数据传到Kafka、gRPC等待arc-storage进行消费。
3. 通过路由解析后，arc-storage开始接收发过来的数据包，进行二次解析。
4. arc-storage可将arc-consmuer发来的数据按数据类型保存为wav格式文件或导入到TDEngine时序数据库，也可对数据进行压缩后存储。
6. 数据量大时，根据实际需要可部署多个arc-storage服务，缓解服务器压力。

### 数据管理功能流程
- arc-consmuer使用gnet连接所有上线的设备，根据已定好的协议实现数据的采集，通过路由把解析好的数据实时发送至kafka消息队列或者连接gRPC服务将数据转发出去。
- 中间件
  - 数据管理服务按顺序消费kafka队列中的数据
  - arc-storage监听连接请求，通过8080端口监听来自客户端的HTTP/2(gRPC)连接。
- gRPC连接成功后，开始实时处理接收到数据，根据协议对数据进行解析，解析后将数据按数据类型分别操作对应的逻辑处理。
- 数据管理服务对外提供可调用的RESTful API，使用web可查询数据库和文件类型的历史数据。
- 结合Prometheus监控系统及Grafana可视化服务，实时监控arc-storage提供的的监控指标(Metrix)。
- 数据管理服务最终的目的是将数据进行落盘和管理，数据落盘主要分一下两部分


## 后端软件结构与功能实现

数据管理服务是基于`Echo Web Framework`开发的后台数据管理服务，集成数据缓存(filecahe)、数据存储(file,tao,HayStack)、
数据接收(`kafka`,`gRPC`)。提供可扩展的`RESTful API`，对已存储的历史数据进行访问等服务。

后端采用多协程实现，包括定时任务相关、数据解析、服务监控、服务监听、Kafka数据消费、gRPC数据处理等等。

数据缓存、接收、解析、存储部分使用的第三方框架或工具如下:

### Echo Web Framework

Echo 是一个高性能，极简Go语言Web框架，主要负责url路由和控制器部分。

使用Echo作为Web框架主要原因
- 路由优化: echo 使用经过优化的 HTTP 路由器，可对路由进行优先级排序
- echo 支持构建健壮且可扩展的 RESTful API
- HTTP/2: echo 支持 HTTP/2
- 中间件: echo 具有许多内置的中间件（一个函数，嵌入在 HTTP 的请求和响应之间），并且支持定义中间件，且可以在根，组或路由级别设置中间件
- 数据绑定: echo 支持将请求数据（JSON，XML 或表单数据）绑定到指定的结构体上
- 数据响应: echo 支持多种格式（JSON, XML, HTML, 文件，附件，内联，流或 Blob）的 HTTP 数据响应
- 可扩展: echo 拥有可定制的集中 HTTP 错误处理和易于扩展的 API 

>代码链接: [Echo Web Framework](https://github.com/labstack/echo)


### Kafka 消息队列

kafka的本质是一个数据存储平台，流平台，只是他在做消息发布，消息消费的时候把他作为消息中间件使用。

基本概念:
- Broker: 消息中间件所在的服务器
- Topic(主题): 每条发布到Kafka集群的消息都有一个类别，这个类别被称为Topic
- Partition(分区): Partition是物理上的概念，体现在磁盘上面，每个Topic包含一个或多个Partition
- Producer: 负责发布消息到Kafka broker
- Consumer: 消息消费者，向Kafka broker读取消息的客户端
- Consumer Group(消费者群组): 每个Consumer属于一个特定的Consumer Group
- offse(偏移量): 是kafka用来确定消息是否被消费过的标识，在kafka内部体现就是一个递增的数字

kafka消息发送的时候,可以先把需要发送的消息缓存在客户端,等到达一定数值时,再一起打包发送,而且还可以对发送的数据进行压缩处理，减少在数据传输时的开销。

Kafka优缺点(gRPC待测试成熟后替换)

有点:
- 基于磁盘的数据存储
- 高伸缩性
- 高性能
- 应用场景:收集指标、日志、流处理

缺点:
- 运维难度大
- 偶尔有数据混乱情况
- 对zookeeper强依赖
- 多副本下对带宽有一定要求

Kafka服务启动配置
```golang
# Topic 为 "arc-consmuer"
"bootstrap.servers":       arc.config.KafkaServer, //kafka集群消费地址
"group.id":                "arc",
"broker.address.family":   "v4",
"fetch.message.max.bytes": arc.config.MessageMaxBytes, //"queued.max.messages.kbytes": 64*1024*1024,
"auto.offset.reset":       "earliest",                   //从提交的offset开始消费,无提交的offset时，从头开始消费
"session.timeout.ms":      6 * 1000,                     //连接超时时间,定位已经挂掉的 Consumer,踢出 Group
"heartbeat.interval.ms":   2000,                         //及时发送心跳避免rebalance
"auto.commit.enable":      true,                         //offset同步到zookeeper,从zookeeper获取最新的offset
"max.poll.interval.ms":    60000,
```


### gRPC 服务

Remote Procedure Call-远程过程调用,使用HTTP2.0

gRPC可以通过protobuf定义接口，有更加严格的接口约束条件。而且protobuf可以将数据序列化为二进制编码，减少传输量。

gRPC通过HTTP2.0使用streaming模式，也就是流式数据处理，可以更快速，更高效处理文件。

gRPC作为一个部件来使用，不适用于大并发的情况，如果部署场景无法满足，需要考虑多实例并行。

gRPC通信有4种请求/响应模式：
- 简单模式(Simple RPC)
- 服务端数据流模式(Server-side streaming RPC)
- 客户端数据流模式(Client-side streaming RPC)
- 双向数据流模式(Bidirectional streaming RPC)

arc-storage使用客户端数据流模式(Client-side streaming RPC)。

客户端源源不断的向服务端发送数据流，而在发送结束后，由服务端返回一个响应。

gRPC服务配置
```golang
//基于Stream的滑动窗口，类似于TCP的滑动窗口，用来做流控，默认64KiB，吞吐量上不去,可以修改为1GB
grpc.InitialWindowSize(arc_grpc.InitialWindowSize),
//于Connection的滑动窗口，默认16 * 64KiB，吞吐量上不去,,可以修改为1GB
grpc.InitialConnWindowSize(arc_grpc.InitialConnWindowSize),
 // 长连接参数,超过KeepAliveTimeout，关闭连接
grpc.KeepaliveParams(kasp),
//kaep-长连接ping 参数如果为true，当连接空闲时仍然发送PING帧监测，如果为false，则不发送忽略。
grpc.KeepaliveEnforcementPolicy(kaep),
// 最大接收    
grpc.MaxRecvMsgSize(arc_grpc.MaxRecvMsgSize),
// 最大发送
grpc.MaxSendMsgSize(arc_grpc.MaxSendMsgSize),
```

>[gRPC 官网](https://grpc.io/)

### RESTful API

>[数据管理服务提供API](机器声纹历史数据访问接口.md)

REST全称是Representational State Transfer，中文意思是表述（编者注：通常译为表征）性状态转移。

REST指的是一组架构约束条件和原则。" 如果一个架构符合REST的约束条件和原则，我们就称它为RESTful架构。

关键概念：
- 资源与URI
- 统一资源接口
- 资源的表述
- 资源的链接
- 状态的转移

详细查看
>[RESTful API概念](https://www.runoob.com/w3cnote/restful-architecture.html)

### Haystack

小文件存储 haystack。

架构比较简单，分为三部份：Haystack Directory, Haystack Cache, Haystack Store。

>相关文档查看 [haystack](https://www.jianshu.com/p/29bd95e5db20)

### FileCache 文件存储

主要用于缓存当前操作的文件类型数据。

arc-storage接收数据后，根据每个采集器ID，进行掩码(`decodeWorkerMask=0x0F`)转换操作，将数据流输入至分配的协程中进行计算。
数据流操作分配16个协程进行逻辑运算。

```golang
# 从kafka接收数据流
arc.decodeJobChans[e.Key[len(e.Key)-1]&decodeWorkerMask] <- e.Value

# 从GRPC接收数据流
arc.decodeJobChans[mes.Key[len(mes.Key)-1]&decodeWorkerMask] <- mes.Value
```

保存流程简述：
- 按照时间设置进行1分钟的数据缓存
- 满足落盘条件，将缓存数据进行落盘
- 清空缓存，开始缓存下一分钟数据

保存操作，包含数据切分方式详细查看[README.md](../README.md)

提供数据保存格式，wav文件或者wav.gz压缩文件

### DataBase 

使用TDengine的模块之一时序数据库。
开发时将减少研发的复杂度、系统维护的难度，TDengine还提供缓存、消息队列、订阅、流式计算等功能，为物联网、工业互联网大数据的处理提供全栈的技术方案，是一个高效易用的物联网大数据平台。

特点：
- 相比于通用数据库有10倍以上的性能提升 
- 存储空间不到通用数据库的1/10
- 将数据库、消息队列、缓存、流式计算等功能融为一体，降低应用开发和维护的复杂度成本
- 数据可在时间轴上或多个设备上进行聚合，具有强大的数据分析功能
- 即可与Telegraf, Grafana, EMQ, HiveMQ, Prometheus, MATLAB, R等集成，与第三方工具无缝连接
- 安装集群简单快捷，无需分库分表，实时备份，零运维成本、零学习成本


>[TDEngine](TDEngine说明文档.md)

### Protobuf

Protobuf实际是一套类似Json或者XML的数据传输格式和规范，用于不同应用或进程之间进行通信时使用。通信时所传递的信息是通过Protobuf定义的message数据结构进行打包，然后编译成二进制的码流再进行传输或者存储。

rotobuf有如下优点:
- 序列化后体积很小:消息大小只需要XML的1/10 ~ 1/3
- 解析速度快:解析速度比XML快20 ~ 100倍
- 多语言支持
- 更好的兼容性,Protobuf设计的一个原则就是要能够很好的支持向下或向上兼容

### Crontab 未完善

主要用于定时拷贝数据

- 数据库数据定时拷贝至指定路径(挂在磁盘)
- 可定时拷贝文件数据至指定路径，同上

该服务需要调用接口启动(暂内部使用)

- 记录拷贝的开始和结束时间，下一次再进行数据拷贝时，先查询数据库，读取上一次拷贝的结束时间
- 再次进行拷贝操作时，从上次结束的时间开始进行拷贝，拷贝记录保存至数据库


### Grafana 系统监控数据可视化界面

```bash
# traefik.http.routers.grafana.rule: "PathPrefix(`/admin/grafana`)"

例如:
192.168.8.244/admin/grafana
账号：admin 密码: admin
```
