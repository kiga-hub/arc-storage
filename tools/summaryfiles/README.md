# 使用说明

该程序用来获取指定目录下，所有传感器不同日期的数据大小。

## 操作方式

1. 使用 root 账号登陆机器，默认会进入 /root 目录。
2. 将程序目录拷贝到数据所在的机器的 /root 目录中。
3. 进入到程序目录，将 client 可执行程序赋予可执行权限： 
    
```bash
chmod +x ./summaryfiles
```

4. 执行该程序，指定传感器数据所在的目录（使用绝对路径），例如：

```bash
# 目录为实际目录，这里仅供参考
./summaryfiles -dir=/arc/local/demonode/arc-storage
```

### 执行结果

程序会将结果写入当前目录下的data-日期时间.txt文件中，日期时间精确到秒，格式：20220413120530

### 文件内容

格式：

```text
传感器目录   总文件数量   总文件大小
日期目录    总文件数量   总文件大小
日期目录    总文件数量   总文件大小
===
传感器目录   总文件数量   总文件大小
日期目录    总文件数量   总文件大小
日期目录    总文件数量   总文件大小
===
```

例如：

```text
A00000000000    5   100MB
20220413        2   11MB
20220414        2   22MB
20220415        1   33MB
===
A00000000002    0   0MB
===
A00000000003    4   200MB
20220415        1   10MB
20220416        3   20MB
===
A00000000004    0   0MB
20220415        0   0MB
===
 ```

### 备注
- 查看参数使用 -h，例：./summaryfiles -h