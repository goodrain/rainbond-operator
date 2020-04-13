添加sqlite后遇到的问题

1、sqlite3运行需要cgo，所以编译镜像时`CGO_ENABLED=1`添加cgo支持
2、添加cgo支持后，golang:1/13镜像启动时会报错
```cassandraql
standard_init_linux.go:190: exec user process caused "no such file or directory"
```
解决办法：

添加编译参数`-extldflags -static`

---
在docker的alpine镜像上运行cgo项目会出现问题，提示panic: standard_init_linux.go:175: exec user process caused "no such file or directory"问题。

原因是当cgo开启时，默认是按照动态库的方式来链接so文件的，但alpine只支持静态链接，所以会出错。

解决方案：

通过设置CGO_ENABLED=0来解决，此时cgo也不可用了。此法不行

调用go build --ldflags "-extldflags -static" ，来让gcc使用静态编译可以解决问题。

或者采用更大的Linux镜像，比如Ubuntu也可以解决。
————————————————

版权声明：本文为CSDN博主「jigetage」的原创文章，遵循 CC 4.0 BY-SA 版权协议，转载请附上原文出处链接及本声明。

原文链接：https://blog.csdn.net/jigetage/article/details/90378490