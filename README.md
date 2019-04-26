# rttys([中文](/README_ZH.md))

[1]: https://img.shields.io/badge/license-LGPL2-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-2.10.0-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://travis-ci.org/zhaojh329/rttys.svg?branch=master
[10]: https://travis-ci.org/zhaojh329/rttys

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
[![Build Status][9]][10]

The server side of [rtty](https://github.com/zhaojh329/rtty)

# Usage
## download the precompiled programs from [Release](https://github.com/zhaojh329/rttys/releases) page according to your os and arch or compile it by yourself.

    go get -u github.com/zhaojh329/rttys

## See Supported Command Line Parameters

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

## run as root (use system credentials)

    sudo ./rttys

## run as normal user (define username and password in config file)

    ./rttys

## View logs when running in the background

    cat /var/log/rttys.log

## Authorization

    ./rttys -gen-token
    Please set a password:******
    Your token is: 34762d07637276694b938d23f10d7164

    ./rttys -token 34762d07637276694b938d23f10d7164

# Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.
