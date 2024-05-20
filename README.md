# MapReduce实现

主要结构按照论文编写：

<p align="center">
    <img src="./imgs/image-20240520003916.png"/>
</p>

包含容错功能，默认10s后未完成任务则重新分配，支持多机部署

分布式文件系统采用hdfs

与论文不同的地方是中间文件也存储于分布式文件系统，减少了额外的RPC调用

其中Reduce数量由代码指定，Map数量由输入文件数量决定

## 代码文件

```
.
├── apps
│   └── wordcount.go // 用户编写部分
├── imgs
├── main
│   ├── input.txt
│   ├── mrmaster.go  // 部署一个Master节点
│   └── mrworker.go  // 部署一个Worker节点
├── mr
│   ├── master.go    // Master逻辑部分
│   ├── rpc.go       // RPC相关定义
│   └── worker.go    // Worker逻辑部分
├── go.mod
├── README.md
└── run.sh
```

主节点与从节点间通过RPC通信，提供三个RPC接口：`RegisterWorker`，`ApplyTask`，`FinishTask`

## 运行

### 单机伪分布式运行：

`sh run.sh`

结果在`main/tmp/answer`中

### 分布式运行：

切换分支：`git checkout multi-machine`

调用了库：`https://github.com/colinmarc/hdfs`

若要编译，使用`go get github.com/colinmarc/hdfs/v2`下载库

然后准备多台机器，其中一台机器做Master，其余机器做Worker，启动hdfs服务

修改`mr/worker.go`中的`masterAddress`常量为主节点地址

编译后，在主节点机器上运行`./mrmaster input.txt`

从节点机器上运行`./mrworker ../apps/wordcount.so`

运行完成后mrmaster进程会打印结果

也可以从hdfs中`/mr/`路径下查看产出结果

再次运行注意删除此路径
