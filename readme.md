A reverse UDP proxy for Teeworlds game play.

Usage: python3 udp_proxy.py \<local port> \<remote ip> \<remote port>

Run this Script in your server. Then you can connect the game server remote_ip:remote:port by your_server_ip:local_port


原理：
假设你的机器为A,游戏服务器为B,A<->B之间通讯质量比较差，丢包率为10%。

现在我建立一个B的反向代理服务器C，可以把A发送到C的数据转发给B，B返回的数据则通过C返回给A，即A<->C<->B

如果B到A和C的通讯质量都比较好(前提条件)，A<->C的丢包率只有1%，B<->C丢包率只有1%，那样的话应该能降低整体丢包率。

用的时候只需要把主机地址这一栏改为我的反向代理服务器的地址