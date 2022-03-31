# tcp-server
 
简单演示json rpc的服务，ue4客户端可以参考[这里](https://github.com/DAN-AND-DNA/TopDownSoulsLike)


1. 裸着写tcp服务
2. 缓存参考muduo那种修改索引实现，不拷贝
3. 减少拷贝，直接处理二进制数据
4. 代码少，性能强