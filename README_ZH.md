# rttys

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-3.1.3-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://travis-ci.org/zhaojh329/rttys.svg?branch=master
[10]: https://travis-ci.org/zhaojh329/rttys

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
[![Build Status][9]][10]

这是[rtty](https://github.com/zhaojh329/rtty)的服务器程序。

# 如何使用
## 从[Release](https://github.com/zhaojh329/rttys/releases)页面下载编译好的二进制文件或者自己编译

    go get -u github.com/zhaojh329/rttys

## 命令行参数

    ./rttys -h
    Usage of rttys:
      -addr-dev string
            address to listen device (default ":5912")
      -addr-user string
            address to listen user (default ":5913")
      -conf string
            config file to load (default "./rttys.conf")
      -gen-token
            generate token
      -http-password string
            password for http auth
      -http-username string
            username for http auth
      -log string
            log file path (default "/var/log/rttys.log")
      -ssl-cert string
            certFile Path
      -ssl-key string
            keyFile Path
      -token string
            token to use
      -white-list string
            white list(device IDs separated by spaces or *)

## 认证

    ./rttys -gen-token
    Please set a password:******
    Your token is: 34762d07637276694b938d23f10d7164

    ./rttys -token 34762d07637276694b938d23f10d7164

## 作为Linux服务运行
移动rttys可执行程序到/usr/local/bin/

    sudo mv rttys /usr/local/bin/

拷贝配置文件到/etc/rttys/

    sudo mkdir /etc/rttys
    sudo cp rttys.conf /etc/rttys/

创建一个systemd单元文件: /etc/systemd/system/rttys.service

    [Unit]
    Description=rttys
    After=network.target

    [Service]
    ExecStart=/usr/local/bin/rttys -conf /etc/rttys/rttys.conf
    TimeoutStopSec=5s

    [Install]
    WantedBy=multi-user.target

要首次启动该服务，请执行通常的systemctl操作:

    sudo systemctl daemon-reload
    sudo systemctl enable rttys
    sudo systemctl start rttys

您可以通过以下方式停止服务:

    sudo systemctl stop rttys

# 贡献代码
如果你想帮助[rttys](https://github.com/zhaojh329/rttys)变得更好，请参考
[CONTRIBUTING_ZH.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING_ZH.md)。
