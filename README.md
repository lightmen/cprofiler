# 概述

cprofiler是一款根据网上[项目](https://github.com/xyctruth/profiler)改造的持续分析系统，该系统用来抓取和分析go程序的profile数据。

该系统支持多种profile样本抓取和分析，包括：`trace` `profile` `mutex` `heap` `goroutine` `allocs` `block` `threadcreate` 。

# 使用

## 配置

在运行cprofiler前，需要先整个配置文件cprofiler.yml，cprofiler的配置参考 [Pyroscope ](https://pyroscope.io/)来设计的，示例内容如下：

```YAML
scrape-configs:
  - job: cprofiler
    interval: 60s         # the interval for scrape profile data
    expiration: 168h      # the profile data expire time
    # enabled-profiles include: profile, mutex, heap, goroutine, allocs, block, threadcreate, trace, all
    # all : means all sample type except trace
    # if not config, the default is all
    enabled-profiles: [all, trace]
    path-profiles:
      profile: /debug/pprof/profile?seconds=10
      trace: /debug/pprof/trace?seconds=10
    target-configs:
      - application: cprofiler
        hosts:
          - 127.0.0.1:9000
        labels:
          env: dev
```

## 运行

### 服务端

使用如下命令运行cprofiler：

```Shell
./cprofiler -config-path ./cprofiler.yml
```

注意： config-path可以不传，不传的话，默认加载 ../conf/cprofiler.yml 配置文件。

运行后，cprofiler会监听 8080 端口，我们可以通过http请求，访问cprofiler提供的请求。



### 前端

通过服务端提供的http接口访问的话，交互性不那么友好。这里，提供一个可视化前端界面来访问cprofiler 的http接口，前端代码运行命令如下：

```Shell
cd ui (注意：ui目录为项目源代码的目录）
npm install --registry=https://registry.npm.taobao.org
npm run dev --base_api_url=http://localhost:8080
```

其中： http://localhost:8080 为cprofiler的监听地址



前端运行后，前端监听的是80端口，通过 localhost:80 可以访问前端界面，界面示例如下：

![img](https://forever9.feishu.cn/space/api/box/stream/download/asynccode/?code=MGFiNGIwNjJjYjViY2RkYzAwNTE0MmRkNDkzZTRhY2NfSXNlVzJTNUFXbW5mb0o2Rk5odzFqdVRJd0U1aENCR0hfVG9rZW46Ym94Y24xbTRIUzVBakk1M1BScE9FbEs5enJmXzE2NTUyNjE1ODU6MTY1NTI2NTE4NV9WNA)





最上面一行，可以选择展示的样本、过滤的label 和 时间



中间的是抓取的样本数据的气泡图，点击气泡可以跳转到具体样本的分析详情。比如，点击 profile_cpu中的气泡，跳转到如下界面：

![img](https://forever9.feishu.cn/space/api/box/stream/download/asynccode/?code=ZDRjNmEwZDA2YjBhMGU2N2RlZTI2OThmN2I3MGJlY2FfbmNJbXkxbklGaU5UQnFLMEVsS05vMkJHbXZuNnBTN2xfVG9rZW46Ym94Y250UmduNWx1Tk44N1VlQlRic1VXNWpmXzE2NTUyNjE1ODU6MTY1NTI2NTE4NV9WNA)







![img](https://forever9.feishu.cn/space/api/box/stream/download/asynccode/?code=NjhiMTMyMmFhMDNlZmQ2MzAxMjhiNmM0Zjg2YTJjMDFfR3NoMVJPb1hEeXVZMzh3NDlocE1iQVVpNWJSSGVqTzJfVG9rZW46Ym94Y25IRlZLeHdsT3RJaHp1OFdnWGRkNkpkXzE2NTUyNjE1ODU6MTY1NTI2NTE4NV9WNA)



# 接口

下面介绍cprofiler服务端提供的http接口

## /api/profile_meta/:sample_type

### 说明

获取某个时间范围内 sample_type 样本的数据

其他sample_type的值通过 `/api/sample_types` 接口获取

### 参数

- start_time: 开始时间，格式RFC3339(yyyy-mm-ddThh:mm:ss.000Z) ， 必填
- end_time: 结束时间， 格式RFC3339，必填
- lbs: 过滤标签，map类型，选填，label的值从`/api/group_labels` 获取
- condition：标签条件，选填，值为 AND 或者 OR， 不填为 AND

### 示例

http://192.168.15.115:8080/api/profile_meta/profile_cpu?&start_time=2022-06-28T16:00:00.000Z&end_time=2022-06-29T16:00:00.000Z&lbs[_app]=pokersrv&lbs[_host]=192.168.15.115:16012

```JSON
[{"TargetName":"192.168.15.115:16012","ProfileMetas":[{"ProfileID":"31","ProfileType":"profile","SampleType":"profile_cpu","JobName":"cashcow_lightmen","Host":"192.168.15.115:16012","App":"pokersrv","SampleTypeUnit":"nanoseconds","Value":4210000000,"Timestamp":1654770439076,"Duration":10108768696,"Labels":[{"Key":"env","Value":"dev"}]},{"ProfileID":"63","ProfileType":"profile","SampleType":"profile_cpu","JobName":"cashcow_lightmen","Host":"192.168.15.115:16012","App":"pokersrv","SampleTypeUnit":"nanoseconds","Value":3880000000,"Timestamp":1654770509282,"Duration":10148589892,"Labels":[{"Key":"env","Value":"dev"}]}]}]
```



## /api/group_sample_types

### 说明

获取样本类型，样本分组组织

### 参数

无

### 示例

http://localhost:8080/api/group_sample_types

```JSON
{"allocs":["allocs_alloc_objects","allocs_alloc_space","allocs_inuse_objects","allocs_inuse_space"],"block":["block_contentions","block_delay"],"goroutine":["goroutine"],"heap":["heap_alloc_objects","heap_alloc_space","heap_inuse_objects","heap_inuse_space"],"mutex":["mutex_contentions","mutex_delay"],"profile":["profile_cpu","profile_samples"],"threadcreate":["threadcreate"]}
```



## /api/sample_types

### 说明

获取样本类型

### 参数

无

### 示例

http://localhost:8080/api/sample_types

```JSON
["allocs_alloc_objects","allocs_alloc_space","allocs_inuse_objects","allocs_inuse_space","block_contentions","block_delay","goroutine","heap_alloc_objects","heap_alloc_space","heap_inuse_objects","heap_inuse_space","mutex_contentions","mutex_delay","profile_cpu","profile_samples","threadcreate"]
```



## /api/group_labels

### 说明

获取所有的label

### 参数

无

### 示例

http://localhost:8080/api/group_labels

```JSON
{"_host":[{"Key":"_host","Value":"192.168.15.115:16012"},{"Key":"_host","Value":"192.168.15.115:16022"},{"Key":"_host","Value":"192.168.15.115:8150"}],"_job":[{"Key":"_job","Value":"lobbysrv"},{"Key":"_job","Value":"pokersrv"}],"env":[{"Key":"env","Value":"dev"}]}
```



## /api/download/:id

### 说明

下载 id 的profile数据

其中 id 的值为` /api/profile_meta/:sample_type` 接口返回的 ProfileID 字段

### 参数

无

### 示例

http://localhost:8080/api/download/692

将会下载profile文件：

暂时无法在文档外展示此内容



## /api/pprof/ui/*

### 说明

利用go tool pprof 可视化分析profile样本

### 参数

无

### 示例

http://localhost:8080/api/pprof/ui/692/?si=cpu

![img](https://forever9.feishu.cn/space/api/box/stream/download/asynccode/?code=OWYwODM1YWViOTJhYjAzYjVmZmJkODFjNTM1MGExNjZfclJ5NGR1VlVocE0ySlpKSm9tUlpmQjlZRW5tclJYZ2VfVG9rZW46Ym94Y245SzB2bktocm40Y3M2b2E2azN2S2RiXzE2NTUyNjE1ODU6MTY1NTI2NTE4NV9WNA)