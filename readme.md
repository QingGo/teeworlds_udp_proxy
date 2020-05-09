https://github.com/QingGo/teeworlds_udp_proxy/workflows/.github/workflows/go.yml/badge.svg

A reverse UDP proxy for Teeworlds game play.

Usage: 
```
python3 udp_proxy.py <local port> <remote ip> <remote port>
```

Run this Script in your server. Then you can connect the game server remote_ip:remote:port by your_server_ip:local_port

The default maximum number of user(inculding dummy) at the same time is 4. It is limit by the Teeworlds server setting "sv_max_clients_per_ip", which default value is 4.

In order to improve performance, I rewrite the python code in golang. After the python version is deployed locally, when the client connects to the local server through a reverse proxy, the in-game ping value is about 24, while the relative go version ping value is only about 3, which greatly improves performance.

You can download the binary build of win64 and linux64 from release page, or you can build it by yourself.

* Golang version Build:
```
go get github.com/sirupsen/logrus
go build udp_proxy.go
````
* Golang version Run:
```
.\udp_proxy.exe <local port> <remote ip> <remote port> (Windows)
.\udp_proxy <local port> <remote ip> <remote port> (Linux)
```

If you want to deploy the golang version proxy into a docker container, you can build the docker image by yourself: 
```
go mod init
go build udp_proxy.go
go mod vendor
docker build -t udp_proxy:1.0.0 .
docker run -d -p <host local port>:<container local port>/udp --name my_udp_proxy udp_proxy:1.0.0 ./udp_proxy <container local port> <remote ip> <remote port>
```
Or you could use the docker image build by me in release page:
```
docker load < ./udp_proxy_docker.tar
docker run -d -p <host local port>:<container local port>/udp --name my_udp_proxy udp_proxy:1.0.0 ./udp_proxy <container local port> <remote ip> <remote port>
```

原理：
假设你的机器为A,游戏服务器为B,A<->B之间通讯质量比较差，丢包率为10%。

现在我建立一个B的反向代理服务器C，可以把A发送到C的数据转发给B，B返回的数据则通过C返回给A，即A<->C<->B

如果C到A和B的通讯质量都比较好(前提条件)，A<->C的丢包率只有1%，B<->C丢包率只有1%，那样的话应该能降低整体丢包率。

用的时候只需要把主机地址这一栏改为我的反向代理服务器的地址

只能同时供4个玩家一起使用（包含dummy），这个受到Teeworlds服务器的sv_max_clients_per_ip的限制。

在部署时需要确认\<local port>端口(用于和client通信)，和[22223, 22800]的端口(随机取若干个用来和server通信)可以通过防火墙收发数据。腾讯云和阿里云需要额外在网页控制台配置。普通linux服务器可以通过改iptables配置。

为了提高性能，把python代码用golang重新写了一遍。python版本部署在本地后，用客户端通过反向代理连接本地服务器时，游戏内ping值为24左右，而相对的go版本ping值仅为3左右，大大提高了性能。