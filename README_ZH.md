# rttys

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-4.4.5-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://github.com/zhaojh329/rttys/workflows/build/badge.svg
[12]: https://deepwiki.com/badge.svg
[13]: https://deepwiki.com/zhaojh329/rttys

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
![Build Status][9]
[![Ask DeepWiki][12]][13]

这是[rtty](https://github.com/zhaojh329/rtty)的服务器程序。

## 从源码构建
golang and node 20+ is required

    cd ui
    npm install
    npm run build
    cd ..
    go build

## 认证(可选)
### Token
一个随机字符串，由用户自己生成，可以被认为是设备连接时的密码.

    $ rttys -t 34762d07637276694b938d23f10d7164

### 使用自己的认证服务器
如果配置了设备 hook url, 设备连接时, rttys 会向此 url 发起请求, 认证服务器返回是否通过认证.

请求数据格式:

    {"devid":"test", "token":"34762d07637276694b938d23f10d7164"}

认证服务器响应 HTTP 200 表示允许设备连接.

### SSL 双向认证(mTLS)
您可以在配置文件中指定设备 CA 存储(有效文件)或在 CLI 中指定设备 CA 存储(参数 ssl-cacert) 来启用 mTLS。
存储中没有有效 CA 的设备将在 TLS 握手中断开连接。

## nginx 反向代理

```
# rttys.conf

addr-user: 127.0.0.1:5913

addr-http-proxy: 127.0.0.1:5914
http-proxy-redir-url: http://web.your-server.com
http-proxy-redir-domain: .your-server.com
```

```
# nginx.conf

server {
    listen 80;

    server_name rtty.your-server.com;

    location /connect/ {
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_pass http://127.0.0.1:5913;
    }

    location / {
        proxy_pass http://127.0.0.1:5913;
    }
}

server {
    listen 80;

    server_name web.your-server.com;

    location / {
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_pass http://127.0.0.1:5914;
    }
}
```

在 rttys.conf 中的参数 http-proxy-redir-url 和 http-proxy-redir-domain 也可以通过在 nginx 中设置新
的 HTTP headers 来配置.

```
server {
    listen 80;

    server_name rtty.your-server.com;

    location /connect/ {
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_pass http://127.0.0.1:5913;
    }

    location /web/ {
        proxy_set_header HttpProxyRedir http://web.your-server.com;
        proxy_set_header HttpProxyRedirDomain .your-server.com
        proxy_pass http://127.0.0.1:5913;
    }

    location / {
        proxy_pass http://127.0.0.1:5913;
    }
}

server {
    listen 80;

    server_name web.your-server.com;

    location / {
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_pass http://127.0.0.1:5914;
    }
}
```

## Docker

    sudo docker run -it -p 5912:5912 -p 5913:5913 -p 5914:5914 zhaojh329/rttys:latest \
        --addr-http-proxy :5914

使用配置文件

    sudo mkdir -p /opt/rttys
    sudo sh -c 'echo "addr-http-proxy: :5914" > /opt/rttys/rttys.conf'
    sudo docker run -it -p 5912:5912 -p 5913:5913 -p 5914:5914 -v /opt/rttys:/etc/rttys \
        zhaojh329/rttys:latest -conf /etc/rttys/rttys.conf

## Star History
[![Star History Chart](https://api.star-history.com/svg?repos=zhaojh329/rttys&type=Date)](https://www.star-history.com/#zhaojh329/rttys&Date)

## 贡献代码
如果你想帮助[rttys](https://github.com/zhaojh329/rttys)变得更好，请参考
[CONTRIBUTING_ZH.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING_ZH.md)。
