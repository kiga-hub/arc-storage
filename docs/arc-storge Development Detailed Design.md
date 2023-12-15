程序系统架构

1. 配置
    - 平台
    - 服务

2. 模块组件注册

3. 数据传输方式(可选)
    - `grpc`: 监听连接请求，通过8080端口监听来自客户端的`HTTP/2(gRPC)`连接
    - `kafka`: 配置默认false

4. 数据处理
    - 解析`Frame` -> `parsedFrame`
    - 数据包处理


5. 存储(冗余操作)
    - 缓存`cache`(singly linked list)
    - 时序数据库(`TDEngine`)
    - 文件系统(haystack,wavfile)

6. SDK生成
    - `swagger`构建自动化文档
    - `client api`

7. 数据查询： `HTTP`协议，自定义`RESTful`接口
    - 实时数据查询(cache)
    - 历史数据查询(file)

## 2.3 流程描述

1. 【流程图】
    ```mermaid
    flowchart LR

    gRPC[gRPC]
    kafka[Kafka]
    arc-storage[arc-storage]
    dispatch[dispatch]
    cache[cache]
    file_cache[file cache]
    restful_api[RESTful API]
    tdengine[TDEngine]
    filesystem[File System]
    haystack[HayStack]
    sdk[SDK API]

    kafka -- Message --> arc-storage
    gRPC -- Data Stream --> arc-storage
    arc-storage -- Decode--> dispatch
    dispatch --Frame --> cache
    dispatch --Frame --> file_cache
    cache -- Search --> restful_api
    file_cache --> tdengine
    file_cache --> filesystem
    file_cache --> haystack
    tdengine -- Search -->restful_api
    filesystem-- Search -->restful_api
    arc-storage -- export --> sdk
    ```

## 2.4 模块描述

1. 【流程图】
    ```mermaid
    flowchart TB
    subgraph Configuration
    Platform_Configure -- Set  --> Service_Configure
    end

    subgraph BasicComponent
    ILogger
    Micro
    TDEngine
    Nacos
    Gossip
    arc-storage
    end

    subgraph DataTransfer
    Decode
    end

    subgraph Storage
    Cache
    HayStack
    FileSystem
    end

    subgraph SDK
    Client_API
    end

    subgraph DataSearch
    RESTful_API
    end

    Service_Configure -- Register --> arc-storage

    arc-storage -- Stream --> Decode
    Decode -- Frame --> Storage

    arc-storage -- Search --> SDK
    arc-storage -- Search --> DataSearch
    ```

### 3.1.1 功能描述

1. 通过`viper`框架在`common`库中初始化一些默认配置，如`logging`，`micro`，`tracing`，`kafka`，`mysql`，`taos`
   等，给其他服务提供默认配置选项，允许其他服务覆盖当前配置，部署环境后，优先使用环境变量，其次使用配置文件，最后使用服务内部配置变量。

2. 通过引用`common`库中拥有默认配置结构体，只需要修改部分配置就能使服务启动，提升开发体验

3. 在项目中是配置相关的代码以*config.go结尾，提升可读性，常量作为key，结构体初始化字段作为value，下一次访问结构体就能拿到对应配置

### 3.1.2 详细设计

#### 3.1.2.1 配置文件(arc-storage.toml)

1. 基本配置
    ```
    [basic]
    zone = "default"
    service = "arc-storage"
    apiport = 80
    apiroot = "/api/data/v1/history"
    prof = false
    inswarm = true
    ```

2. arc-storage存储服务配置
    ```
    [arc]
    dataPath = "/home/arc-storage/data"
    ...
    ```

3. kafka配置
    ```
    kafkaenable = false
    kafkaServer = "localhost:9092"
    ...
    ```

4. taos配置
    ```
    [taos]
    taosenable = true
    host = "taos"
    ```

5. gRPC配置
    ```
    [grpc]
    grpcenable = true
    grpcserver = ":8080"
    ```

6. log 配置
    ```
    [log]
    level = "INFO"
    ```

7. pprof 性能分析配置
    ```
    [pprof]
    pprofenable = false
    ```

8. trace 链路追踪配置
    ```
    [trace]
    jaegerCollector = "http://tempo:14268"
    tempoQuery = "tempo:9095"
    ```

#### 3.1.2.2 代码实现

1. 代码目录结构
    ```
    |-arcStorage
        |-pkg  
            |-config 
                |-cacheConfig.go 缓存配置
                |-arcConfig.go 服务配置
    ```


## 3.2 服务组件注册

### 3.2.1 功能描述

1. 通过`common`库封装`Micro`组件，让所有服务都有统一的注册流程，同时提供启动服务，获取服务状态，关闭服务等方法。只要把服务都抽象成micro组件，就能使用上面所有方法，大大提高程序内聚力

### 3.2.2详细设计

#### 3.2.2.1 配置文件

1. 无

#### 3.2.2.2 代码实现

1. 代码目录结构
    ```
    |-arc-storage
        |-cmd  
            |-root.go （Cobra CLI命令行库）
            |-server.go （基础组件注册，Micro服务初始化）

    |-arc-storage
        |-pkg 
            |-component
                |-component.go arc-storage 相关服务）
    ```

2. 主要方法
    - 基础组件
        - `Gossip` Component: 周期性的散播节点之间设备信息(Sensor ID)
        - `ILogger` Component: 日志组件
        - `Nacos` Component: 服务注册中心、分布式配置中心组件
        - `Micro` Empty Component: 基本服务组件
        - `TDEngine` Component: 涛思时序数据库组件
        - `ArcStorage` Component: 自定义数据存储组件
        - -~~~`gRPC`~~~
        - -~~~`kafka`~~~
        ```golang
        // 服务注册
        server, err := micro.NewServer(
                AppName,
                AppVersion,
                []micro.IComponent{
                    &basicComponent.TaosComponent{},
                    &component.ArcStorageComponent{}, 
                    ...
                }
        )

        // 服务初始化
        err = server.Init()

        // 启动
        err = server.Run()
        ```

    - arc-storage 组件
        ```golang
        // PreInit . config.SetDefaultConfig() 基本配置
        func (c *ArcStorageComponent) PreInit(ctx context.Context) error {}

        // 服务初始化 & 启动
        func (c *ArcStorageComponent) Init(server *micro.Server) error {
            // server.GetElement(&micro.NacosClientElementKey).(*configuration.NacosClient) // 获取组件
            ...
        }
        ```


## 3.3 数据传输

### 3.3.1 功能描述

1. `arc-storage` 开启`gRPC`服务，接收`Receiver`发送过来的数据,解析后传到其他模块使用。

### 3.3.2 详细设计

1. `gRPC`服务有更加严格的接口约束条件。将数据序列化为`Frame`结构二进制编码，减少传输量。

#### 3.3.2.1 配置文件

1. gRPC 配置
    ```
    [grpc]
    grpcenable = true
    grpcserver = ":8080"
    ```
2. kafka 配置
    ```
    [kafka]
    kafkaenable = false
    kafkaServer = "localhost:9092"
    submitCapacity = 16
    submitRequests = 8192 
    submitTimeout = 300
    messageMaxBytes = 67108864
    ```
#### 3.3.2.2 代码实现

1. 代码目录结构
    ```
    |-arc-storage
        |-pkg
            |-grpc
                |-grpc.go (gRPC服务相关配置)
                |-protostream.go （接口）
            |- kafka
                |-kafka.go (kafka服务相关配置)
    ```

2. 主要方法(gRPC)
    - 协议:`proto3`语法构造要传输的数据结构,定义消息类型，保存于frame.proto文件中。
        ```protocol
        syntax = "proto3";
        package proto;

        option go_package = "pb";

        service FrameData {
            rpc FrameDataCallback(stream FrameDataRequest) returns (FrameDataResponse){}
        }

        message FrameDataRequest {
            bytes key = 2;
            bytes value = 3;
        }

        message FrameDataResponse {
            bool successed = 1;
        }
        ```

    - 接口:`protocol buffer`编译器生成对应接口代码
        ```protocol
        type FrameDataClient interface {
            FrameDataCallback(ctx context.Context, opts ...grpc.CallOption) (FrameData_FrameDataCallbackClient, error)
        }
        ```

    - 传输模式:`gRPC`使用客户端数据流模式(`Client-side streaming RPC`)可以更快速，更高效处理多种数据类型文件。
        >设备传感器ID号为`Key`值，数据为`Value`，根据设备ID进行取模操作，分布到对应的协程中进行处理。
        ```golang
        case mes := <-g.grpcmessage:
            isStop = mes.IsStop
            if len(mes.Key) > 0 {
            decodeJobChans[mes.Key[len(mes.Key)-1]&decodeWorkerMask] <- mes.Value
            }
        ```

3. 主要方法(kafka)

    -~~~`kafka`~~~
    - -~~~暂时未使用~~~

## 3.4 数据处理


### 3.4.1 功能描述

1. 解析传过来的二进制数据。解析成Frame结构:(二进制数据解析为Frame结构体，获取协议中的各个字段)。
    - 传感器ID
    - ..

2. dispatch解析后的数据(goroutine)
    - cache
    - filecache
    - TDEngien
    - -~~~Haystack~~~

3. 传感器数据属性判断
    - 时间戳错误
    - SensorID错误
    - 采样率改变
    - 固件信息改变
    - 版本信息改变
    - 数据包不连续
    - 解析错误

4. 数据接受超时处理
    - //数据脏读脏写 TODO // 乔怡轩

5. 异常退出时，缓存数据处理

## 3.4.2 详细设计

#### 3.4.2.1 配置文件

1. 无

#### 3.4.2.2 代码实现

1. 代码目录结构
    ```
    |-arc-storage
        |-pkg
            |-handle.go (数据处理)
            |-decode.go (解析)
            |-timeout.go (超时、异常退出)
    ```
2. 主要方法
    - arc-storage数据帧结构
        ```golang
        // parsedFrame 帧数据
        type parsedFrame struct {
            timestamp       time.Time
            idString        string
            filenamesuffix  string
            sensorType      string
            id              []byte
            dataToStore     []byte
            protocolType    int
            dataToStoreSize int
            idUint64        uint64
            isInterrupt     bool
        }
        ```
    - TCP包结构
        ```golang
        type Frame struct {
            Head      [4]byte //4 头标志 0xFC 0xFC 0xFC 0xFC
            Size      uint32  //4 包大小 [Timestamp, End] = 32+n BigEndian
            Timestamp int64   //8 时间戳/序号 精确到毫秒 BigEndian
            ID        [6]byte //6 设备编号 (Mac Address)
            DataGroup DataGroup //n 数据
            Crc uint16 //2 校验位 [Timestamp, Data], CRC-16 BigEndian
            End byte   //1 结束标志 0xFD
        }
        ```
    - 数据处理方式：
        1. 解析:
            - 由common库提供的方法进行包解析，获取数据
            - 数据处理以数据包中的时间戳为准，对包中的时间戳只进行计算，不更改
            - `parsedFrame`解析错误，丢弃当前数据包
        2. dispatch:
            - 16个协程进行Frame的处理
            - 解析前的数据`protocols.Frame`实时缓存到(`DataCacheContainer`)
            - 解析后数据根据类型存到filecache`Buffer`中
            - 解析的数据保存到TDEngine
        3. 数据包判断:
            - 时间戳:是否缓存，落盘，丢弃
            - 数据属性：采样率、固件版本、传感器ID、数据包是否连续
        4. 落盘:
            - 满足落盘时间(每分钟)对缓存数据进行落盘操作
            - 数据包不连续，超时，异常退出时对已缓存数据进行落盘
    - 代码实现
        ```golang 
        // decodeWorkerMask = 0x0F,根据SensorID取模，由对应的goroutine进行数据处理
        // decodeWorkerCount = 16 协程数
        // dispatch goroutine
        decodeJobChans[mes.Key[len(mes.Key)-1]&decodeWorkerMask] <- mes.Value
        // 16个协程并发处理接收的数据，并解析
        arc.decodeWorker(arc.decodeJobChans[i], arc.decodeResultChans[i])
        output <- arc.decode(j)
        // 处理解析后的数据
        func (arc *ArcStorage) handleDecodeResult(sigchan chan os.Signal, workindex int)
        ```

TODO:
>
> 结合`Prometheus`监控系统及`Grafana`可视化服务，实时监控arc-storage提供的的监控指标(`Metrics`)。
>
> 考虑接收到的数据是否有超时，单开一个协程进行超时判断. // TODO 数据覆盖？
>
> 解析过程阻塞，影响cache的实时性？


## 3.5 存储模块设计文档

### 3.5.1 功能描述

1. 对接收到的数据进行落盘操作，对不同数据类型，保存到不同的介质当中

### 3.5.2 详细设计

1. -~~~原始数据保存至`Haystack`~~~
    - -~~~数据解析后，按时序进行存储~~~

2. arc-storage需要将收集到的数据落地，保存文件格式为[x]


3. 缓存实时数据到cache
    - arc-storage接收到的设备帧数据缓存到内容中供api实时查询使用

4. Arc数据保存到TDEngine

5. 保存接口访问记录到TDEngine

#### 3.5.2.1 配置文件

1. cache配置
    ```
    [cache]
    enable = true
    timeoutmin = 2
    expirems = 120000
    search = true
    ```

2. 数据落盘配置
    ```
    [arc]
    dataPath = "/home/arc-storage/data"  //文件路径
    savetype = 0                      //0默认不压缩
    debugmod = 0
    duration = "day"
    ```

#### 3.5.2.2 代码实现

1.  模块目录结构
    ```
    |-arc-storage
        |-pkg
            |-cache       (缓存模块)
            |-tdengine    (时序数据库)
            |-filecache   (文件系统)
            |-storage      (haystack小文件存储系统)
    ```

2. 主要方法(cache缓存模块):
    - cache结构
        ```golang
        type Cache struct {
            Container   *cache.DataCacheContainer
            ASamplerate uint16
        }
        type ArcData struct {
            Data []byte
        }
        ```
    
    - 缓存结构
        ```
        |-cache对象
            |-Container.caches   (并发安全map,key=采集器id,value=data)
            |-data (帧数据链表结构,每帧数据自带时间,操作时上锁保护,在启动协程中轮询删除过期帧数据)
        ``` 
        - 创建:arc-storage初始化时创建缓存对象
        - 启动:arc-storage启动服务时启动缓存对象,开启一个协程每秒轮询删除过期帧数据
        - 数据输入:grpc模块在收到数据时,调用cache模块存入数据。
        - 数据输出:api根据时间查询数据,如果时间在expirems配置时间内,将从缓存内取出设备帧数据
        - 释放：arc-storage退出时关闭cache对象,退出轮询协程

3. 主要方法(filecache 文件存储):
    - filecache 结构
        ```golang
        // ArcVolumeCache -
        type ArcVolumeCache struct {
            DataCache *sync.Map
            lock      util.MultiLocker
            logger    logging.ILogger
            Cachechan chan *BigFile
            once      sync.Once
        }

        // ArcVolume -
        type ArcVolume struct {
            SensorID        string
            Buffer          *bytes.Buffer
            Dir             string
            CreateTime      time.Time
            SaveTime        time.Time
            FirmWare        string
            StatusOfStorage string
            MinuteStr       string
            Version         string
            Type            string
            SampleRate      float64
            DynamicRange    byte
            Resolution      byte
            Channel         byte
            Bits            byte
            isInterrupt    bool
        }
        ```
        
    - filecache对象结构
        ```
        |-filecache对象
            |-DataCache   (并发安全map,key=采集器id,value=BigFile)
                    |-BigFile (帧数据缓存结构,将接收到的帧数据拼接到buff缓存,记录头文件信息)
        ``` 

    - 数据流程 
        | grpc接收协程| 文件解码和处理协程| filecache协程 | 
        | ---- | ---- | ---- | 
        | 数据按设备id负载均衡,投递给16个解码和文件处理协程，帧数据缓存入对应的filecache对象,如果缓存时间>=1分钟，则将该数据投递到文件存档协程并清空该设备id缓存| 将投递过来的文件数据存入磁盘wav文件，如果存在则采用追加写入|

        - 创建:arc-storage初始化时创建文件对象
        - 启动:arc-storage启动服务时创建16个解码和文件处理协程,处理协程根据设备id号负载均衡配置任务,filecache对象创建文件存档协程
        - 数据输入:
            - 1.grpc收到的帧数据,投递到解码协程，
            - 2.解码协程解码后再投递到数据处理协程
            - 3.数据处理协程将帧数据加入对应的filecache缓存。如果该设备id缓存时间达到1分钟，则将该设备id当前的缓存投递到文件存档协程并清空该设备id的缓存数据
            - 4.文件存档协程将收到的缓存数据存入磁盘wav文件
        - 数据输出:api根据时间查询数据,如果时间在expirems配置时间外,将磁盘查找对应的wav文件。
            - wav文件名包含创建时间,根据创建时间和数据量偏移找出需要查询的数据段，返回给api
        - 释放：arc-storage退出时关闭filecache对象,退出协程
    - wav文件格式
        - wav文件文件格式:采用44字节头信息+byte数据流的二进制文件格式
        - wav文件文件追加:直接将采集到的二进制数据追加到已有的wav文件中,并修改文件名的结束时间
        - wav文件文件命名:采集设备ID_类型_开始时间_结束时间_采样率_固件版本_硬件版本_标志位_arc-storage版本号.wav
            - 结束时间是依照采集数据速度来计算,比如采样率32000，channel=1,位深16的采集设备 每毫秒秒采集存档数据=1000*32000*1*16/8,结束时间=开始时间+(文件大小-44)/每毫秒秒采集存档数据
            - 历史数据查询根据wav文件名开始时间和结束查找到对应的wav文件,根据采集速度来偏移出wav文件中指定时间段的数据

1. 主要方法(TDEngine时序数据库):
    - 相关数据表和超级表:
        ```golang
            // ArcTableName -
            ArcTableName = "arc_v1_"
            TDEngineBackUpSTableName = "taosbaks_v1"
            // TDEngineBackUpTableName -
            TDEngineBackUpTableName = "taosbak_v1"
            // AsyncBackUpSTableName -
            AsyncBackUpSTableName = "asyncbaks_v1"
            // AsyncBackUpTableName -
            AsyncBackUpTableName = "asyncbak_v1"
            // RecordBackUpSTableName -
            RecordBackUpSTableName = "record_s"
            // RecordBackUpTableName -
            RecordBackUpTableName = "record"
            // RecordStatusSTableName -
            RecordStatusSTableName = "status_s"
        ```
    - 代码实现:
        ```golang
        // SearchArcHistoryData query data
        func (c *TaoClient) SearchArcHistoryData(databasename, sensorid string, from, to string, israwdata bool, interval int, fill, function string) (int64, []ArcDataInfo, error) 
        ```

2. 主要方法(HayStack):
    - haystack结构
        ```
        |---storage
            |---backend
            |---idx
            |---needle
            |---needlemap
            |---stats
            |---superblock
            |---types
        ```

    -  核心组件:
        - `Haystack Directory`
            - 生成FID，维护`logical volume`与`physical volume`映射关系

        - `Haystack Cache`
            - `FID` 采用`一致性hash`算法保存

        - `Haystack Store`
            - 最终落地存储服务
            - 内存中维护图片在文件中的`Offset`和`Size`的索引信息。

    -  存储格式:
        - `Store File`
            - 文件头为`Superblock`保存全局信息每个数据段为`Needle`结构，顺序追加到文件尾
            - `Needle`保存数据的详细信息
                - `Cookie`
                - `Key`
                - `Flags`
                - `Size`
                - `Data`
                - `CheckSum`
            - <img src="images/store_file.png" width="60%">

        - `Index File`
            - 加载文件，利用`Index File`快速重建数据索引信息，没有索引文件则顺序扫描`Store File`重建。
            - <img src="images/index_file.png" width="60%">

        - 数据存储目录:每个传感器ID对应一个目录，子目录包含以日期为名的文件夹，接收数据后创建以时间单位的Volume文件。
            - /arc/local/demonode/arc-storage/A00000000000/20220324/arc/
            - 数据段保存时间戳，采样率等信息
            - 数据属性变更(采样率，校验，版本)，新建Volume
            - 每个时间段自动生成保存数据的文件(.dat).和用于检索数据文件的索引文件(.idx)。
            - 数据加载模式:
                - 内存加载(.idx)
                - 数据库文件加载(.ldb)

## 3.6 SDK生成

| 版本  | 修改内容 | 作者  | 时间 |
|-----|------|-----|------------|
| 1   | 初始版本 | 潘雄  | 2022-03-24 |

### 3.6.1 功能描述

1. 对外提供软件开发工具包

### 3.6.2 详细设计

#### 3.6.2.1 配置文件

1. Makefile
    ```
    swagger-json:
        rm -f ./swagger.json
        wget http://127.0.0.1/api${API_ROOT}/swagger/swagger.json

    swagger-client:
        docker pull quay.io/goswagger/swagger:v0.28.0
        docker run --rm -v `pwd`:/src quay.io/goswagger/swagger:v0.28.0 \
        generate client -t /src -f /src/swagger.json --skip-validation \
        --client-package=api/client --model-package=api/models

    swagger-server:
        docker pull quay.io/goswagger/swagger:v0.28.0
        docker run --rm -v `pwd`:/src quay.io/goswagger/swagger:v0.28.0 \
        generate server -t /src -f /src/swagger.json --skip-validation \
        --server-package=api/server --model-package=api/models --main-package=test-server
    ```

#### 3.6.2.2 代码实现

1. 代码: 无

2. 主要方法：
    - swagger构建自动化文档:
        - 为了方便api文档输出，在echo上集成swagger组件，只需要在定义接口的时候提供参数，编译出swagger所需文件。
        - 在启动echo的时候指定swagger文件所在目录，然后通过echoswagger加载路径就可以在网页上访问接口文档了。
        - URL: (example)
            - > http://192.168.8.245/api/data/v1/history/swagger
    - SDK 生成: 
        - 获取arc-storage提供RESTful api
        - wget http://127.0.0.1/api${API_ROOT}/swagger/swagger.json
        - 构建对外提供的接口 arc-storage server client
            ```
            swagger-client:
                docker pull swagger:v0.28.0
                docker run --rm -v `pwd`:/src swagger:v0.28.0 \
                generate client -t /src -f /src/swagger.json --skip-validation \
                --client-package=api/client --model-package=api/models
            ```


## 3.7 数据查询

| 版本  | 模版版本 | 修改内容 | 作者  | 时间 |
|-----|------|------|-----|------------|
| 1   | 1    | 初始版本 | 乔怡轩 | 2022-03-24 |

### 3.7.1 功能描述

1. HTTP数据查询

- [当前版本-数据访问接口](接口文档说明.md)


#### 3.7.3.2 描述

1. 增加一个根据接口名+参数名（字典序）制作签名的函数，叫做**请求签名函数**。
2. 每个查询接口在handle after中间件中，通过**请求签名函数**获取签名，并记录签名和数据大小、查询时间等信息（可用kv数据库存储，加入lru策略，比如只保留最近10次）。
3. 每个查询接口在handle before中间件中，通过confirm参数判断做什么事情:
    * 不带confirm参数，通过签名函数获取该次请求的签名，从kv数据库查询历史情况数据，做最大最小平均值计算，将结果返回给前端，没有查询到数据则返回无参考数据。
    * 带了confirm参数，执行真正的查询操作，并返回数据。
4. 考虑不同项目根据配置文件来定义单次查询的最早时间点、最长执行时间、最大datasize，查询接口：
    * 根据配置文件中的最早时间点做参数判断。
    * 根据配置文件中的最大查询耗时做数据查询超时控制（需要数据库支持），超时报错。
    * 根据配置文件中的最大datasize，与当前签名的历史最大数据尺寸数据做比较，超过则不执行查询，报错提示，当前签名没有历史数据则直接查询。
