# rttys([中文](/README_ZH.md))

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

This is the server program of [rtty](https://github.com/zhaojh329/rtty)

# Usage
## download the pre-built release binary from [Release](https://github.com/zhaojh329/rttys/releases) page according to your os and arch or compile it by yourself.

    go get -u github.com/zhaojh329/rttys

## Command Line Parameters

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

## Authorization

    ./rttys -gen-token
    Please set a password:******
    Your token is: 34762d07637276694b938d23f10d7164

    ./rttys -token 34762d07637276694b938d23f10d7164

## Running as a Linux service
Move the rttys binary into /usr/local/bin/

    sudo mv rttys /usr/local/bin/

Copy the config file to /etc/rttys/

    sudo mkdir /etc/rttys
    sudo cp rttys.conf /etc/rttys/

Create a systemd unit file: /etc/systemd/system/rttys.service

    [Unit]
    Description=rttys
    After=network.target

    [Service]
    ExecStart=/usr/local/bin/rttys -conf /etc/rttys/rttys.conf
    TimeoutStopSec=5s

    [Install]
    WantedBy=multi-user.target

To start the service for the first time, do the usual systemctl dance:

    sudo systemctl daemon-reload
    sudo systemctl enable rttys
    sudo systemctl start rttys

You can stop the service with:

    sudo systemctl stop rttys

# Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.
