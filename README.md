# rttys([中文](/README_ZH.md))

![](https://img.shields.io/badge/license-GPLV3-brightgreen.svg?style=plastic "License")

The server side of [rtty](https://github.com/zhaojh329/rtty)

`Keep Watching for More Actions on This Space`

![](/rtty.svg)

![](/rtty.gif)

# How to install
Install the GO language environment (if you haven't installed it)

	sudo apt-get install golang

Install dependent packages

    go get github.com/gorilla/websocket
    go get github.com/rakyll/statik

Install rtty server

	go get github.com/zhaojh329/rttys

Manual run

	$GOPATH/bin/rttys -port 5912

Install the automatic boot script

    cd $GOPATH/src/github.com/zhaojh329/rttys
    sudo ./install.sh
    sudo /etc/init.d/rttys start

# Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.