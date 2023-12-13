## TDEngien 说明文档

本文档说明 TAOS SQL 支持的语法规则、主要查询功能、支持的 SQL 查询函数，以及常用技巧等内容。

### 支持的数据类型

TDEngine 最重要的是时间戳，插入、查询历史记录，都需要指定时间戳

当前使用版本version: 2.0.14.0，时间格式为 YYYY-MM-DD HH:mm:ss.MS，默认时间分辨率为毫秒

TDengine 缺省的时间戳是毫秒精度，但通过在 CREATE DATABASE 时传递的 PRECISION 参数就可以支持微秒和纳秒。
- Version2.1.5.0之后支持纳秒

```golang
precision 'us'
```

TDengine 对 SQL 语句中的英文字符不区分大小写，自动转化为小写执行。因此用户大小写敏感的字符串及密码，需要使用单引号将字符串引起来。

### SQL 函数

TDEngine支持对数据的聚合和选择查询

聚合函数:
- COUNT: 统计表/超级表中记录行数或某列的非空值个数。
    - SELECT COUNT(*), COUNT(TEMPERATURE) FROM TEMPERATURE;
- AVG: 统计表/超级表中某列的平均值。
    - SELECT AVG(X), AVG(Y), AVG(Z) FROM VIBRATE;
- SUM: 统计表/超级表中某列的和。
    - SELECT SUM(X), SUM(Y), SUM(Z) FROM VIBRATE;
- STDDEV: 统计表中某列的均方差
    - SELECT STDDEV(TEMPERATURE) FROM TEMPERATURE;

选择函数：
- MIN: 统计表/超级表中某列的值最小值。
    - SELECT MIN(X), MIN(Y) FROM VIBRATE;
- MAX: 统计表/超级表中某列的值最大值。
    - SELECT MAX(X), MAX(Y) FROM VIBRATE;
- FIRST: 统计表/超级表中某列的值最先写入的非NULL值。
    - SELECT FIRST(*) FROM TEMPERATURE;
- LAST: 统计表/超级表中某列的值最后写入的非NULL值。
    - SELECT LAST(*) FROM TEMPERATURE;
- TOP: 统计表/超级表中某列的值最大 k 个非 NULL 值。如果多条数据取值一样，全部取用又会超出 k 条限制时，系统会从相同值中随机选取符合要求的数量返回。
    - SELECT TOP(X, 3) FROM VIBRATE;
- BOTTOM: 统计表/超级表中某列的值最小 k 个非 NULL 值。
    - SELECT BOTTOM(X, 2) FROM VIBRATE;
- PERCENTILE: 统计表中某列的值百分比分位数。
    - SELECT PERCENTILE(X, 20) FROM VIBRATE;
- APERCENTILE: 统计表/超级表中某列的值百分比分位数，与PERCENTILE函数相似，但是返回近似结果。
    - SELECT APERCENTILE(X, 20) FROM VIBRATE;
- LAST_ROW: 返回表/超级表的最后一条记录。
    - SELECT LAST_ROW(X) FROM VIBRATE;

### 按窗口切分聚合

TDengine 支持按时间段等窗口切分方式进行聚合结果查询，比如温度传感器每秒采集一次数据，但需查询每隔 10 分钟的温度平均值。这类聚合适合于降维（down sample）操作。

```mysql
SELECT function_list FROM tb_name
  [WHERE where_condition]
  [SESSION(ts_col, tol_val)]
  [STATE_WINDOW(col)]
  [INTERVAL(interval [, offset]) [SLIDING sliding]]
  [FILL({NONE | VALUE | PREV | NULL | LINEAR | NEXT})]
```
>function: 在聚合查询中，function_list 位置允许使用聚合和选择函数，并要求每个函数仅输出单个结果
>> - COUNT
>> - AVG
>> - SUM
>> - STDDEV
>> - LEASTSQUARES
>> - PERCENTILE
>> - MIN
>> - MAX
>> - FIRST
>> - LAST
>> - 而不能使用具有多行输出结果的函数
>>   - TOP
>>   - BOTTOM
>>   - DIFF
>>   - 以及四则运算

>interval: 时间窗口：聚合时间段的窗口宽度由关键词 INTERVAL 指定
>> - 最短时间间隔 10 毫秒（10a）

>fill: 指定某一窗口区间数据缺失的情况下的填充模式
>> - 不进行填充：NONE（默认填充模式）
>> - VALUE 填充：固定值填充，此时需要指定填充的数值。例如：FILL(VALUE, 1.23)
>> - PREV 填充：使用前一个非 NULL 值填充数据。例如：FILL(PREV)
>> - NULL 填充：使用 NULL 填充数据。例如：FILL(NULL)
>> - LINEAR 填充：根据前后距离最近的非 NULL 值做线性插值填充。例如：FILL(LINEAR)
>> - NEXT 填充：使用下一个非 NULL 值填充数据。例如：FILL(NEXT)


TIPS:
- 使用 FILL 语句的时候可能生成大量的填充输出，务必指定查询的时间区间。针对每次查询，系统可返回不超过 1 千万条具有插值的结果
- 在时间维度聚合中，返回的结果中时间序列严格单调递增
- 如果查询对象是超级表，则聚合函数会作用于该超级表下满足值过滤条件的所有表的数据。如果查询中没有使用 GROUP BY 语句，则返回的结果按照时间序列严格单调递增；如果查询中使用了 GROUP BY 语句分组，则返回结果中每个 GROUP 内不按照时间序列严格单调递增

### TAOS SQL 边界限制

- 数据库名最大长度为 32。
- 表名最大长度为 192，每行数据最大长度 16k 个字符（注意：数据行内每个 BINARY/NCHAR 类型的列还会额外占用 2 个字节的存储位置）。
- 列名最大长度为 64，最多允许 1024 列，最少需要 2 列，第一列必须是时间戳。
- 标签名最大长度为 64，最多允许 128 个，可以 1 个，一个表中标签值的总长度不超过 16k 个字符。
- SQL 语句最大长度 65480 个字符，但可通过系统配置参数 maxSQLLength 修改，最长可配置为 1M。
- SELECT 语句的查询结果，最多允许返回 1024 列（语句中的函数调用可能也会占用一些列空间），超限时需要显式指定较少的返回数据列，以避免语句执行报错。
- 库的数目，超级表的数目、表的数目，系统不做限制，仅受系统资源限制。

### Arc-storage历史数据获取接口参数说明

arc-storage查询使用TDEngine查询历史数据

如果不适用聚合查询，即不输入，function、interval、fill这三个参数，则返回from-to时间段内的所有数据，不会进行任何函数处理。

如果输入了上述三个参数中的值，缺省值使用如下默认值
- function：FIRST
- fill: PREV
- interval: 100
