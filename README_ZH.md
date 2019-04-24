# rttys

[1]: https://img.shields.io/badge/license-LGPL2-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-2.9.2-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://travis-ci.org/zhaojh329/rttys.svg?branch=master
[10]: https://travis-ci.org/zhaojh329/rttys

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
[![Build Status][9]][10]

[rtty](https://github.com/zhaojh329/rtty)的服务端。

# 如何使用
## 从[Release](https://github.com/zhaojh329/rttys/releases)页面下载编译好的程序或者自己编译

## 查看支持哪些命令行参数

    ./rttys -h
    Usage of ./rttys:
      -addr string
            address to listen (default ":5912")
      -conf string
            config file to load (default "./rttys.conf")
      -gen-token
            generate token
      -ssl-cert string
            certFile Path (default "./rttys.crt")
      -ssl-key string
            keyFile Path (default "./rttys.key")
      -token string
            token to use

## 以root用户运行(使用系统用户名和密码)

    sudo ./rttys

## 以普通用户运行(用户名和密码来自配置文件)

    ./rttys

## 如何在后台运行模式下查看日志

    cat /var/log/rttys.log

## 认证

    ./rttys -gen-token
    34762d07637276694b938d23f10d7164

    ./rttys -token 34762d07637276694b938d23f10d7164

# 贡献代码
如果你想帮助[rttys](https://github.com/zhaojh329/rttys)变得更好，请参考
[CONTRIBUTING_ZH.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING_ZH.md)。
