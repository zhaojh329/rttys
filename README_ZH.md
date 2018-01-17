# rttys

![](https://img.shields.io/badge/license-GPLV3-brightgreen.svg?style=plastic "License")

[rtty](https://github.com/zhaojh329/rtty)的服务端。

`请保持关注以获取最新的项目动态`

![](/rtty.svg)

![](/rtty.gif)

# 如何安装
安装GO语言环境（如果您还未安装）

    sudo apt-get install golang

安装依赖包

    go get github.com/gorilla/websocket
    go get github.com/rakyll/statik

安装rtty server

    go get github.com/zhaojh329/rttys

手动运行

    $GOPATH/bin/rttys -port 5912

安装自启动脚本，后台运行

    cd $GOPATH/src/github.com/zhaojh329/rttys
    sudo ./install.sh
    sudo /etc/init.d/rttys start

# 贡献代码
如果你想帮助[rttys](https://github.com/zhaojh329/rttys)变得更好，请参考
[CONTRIBUTING_ZH.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING_ZH.md)。

# 技术交流
QQ群：153530783