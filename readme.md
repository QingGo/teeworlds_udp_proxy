A reverse UDP proxy for Teeworlds game play.

Usage: python3 udp_proxy.py \<local port> \<remote ip> \<remote port>

Run this Script in your server. Then you can connect the game server remote_ip:remote:port by your_server_ip:local_port

The default maximum number of user(inculding dummy) at the same time is 4. It is limit by the Teeworlds server setting "sv_max_clients_per_ip", which default value is 4.


原理：
假设你的机器为A,游戏服务器为B,A<->B之间通讯质量比较差，丢包率为10%。

现在我建立一个B的反向代理服务器C，可以把A发送到C的数据转发给B，B返回的数据则通过C返回给A，即A<->C<->B

如果C到A和B的通讯质量都比较好(前提条件)，A<->C的丢包率只有1%，B<->C丢包率只有1%，那样的话应该能降低整体丢包率。

用的时候只需要把主机地址这一栏改为我的反向代理服务器的地址

只能同时供4个玩家一起使用（包含dummy），这个受到Teeworlds服务器的sv_max_clients_per_ip的限制。

在部署时需要确认\<local port>端口(用于和client通信)，和[22223, 22800]的端口(随机取若干个用来和server通信)可以通过防火墙收发数据。腾讯云和阿里云需要额外在网页控制台配置。普通linux服务器可以通过改iptables配置。