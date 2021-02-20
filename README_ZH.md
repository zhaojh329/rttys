# rttys

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-3.4.0-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://github.com/zhaojh329/rttys/workflows/build/badge.svg

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
![Build Status][9]

这是[rtty](https://github.com/zhaojh329/rtty)的服务器程序。

# 如何使用
## 从[Release](https://github.com/zhaojh329/rttys/releases)页面下载编译好的二进制文件或者自己编译

    git clone https://github.com/zhaojh329/rttys

    cd frontend
    npm install
    npm run build
    cd ..

    ./build.sh linux amd64

## 认证
### Token
生成一个 token

    $ rttys token
    Please set a password:******
    Your token is: 34762d07637276694b938d23f10d7164

使用 token

    $rttys -t 34762d07637276694b938d23f10d7164

### SSL 双向认证(mTLS)
您可以在配置文件中指定设备 CA 存储(有效文件)或在 CLI 中指定设备 CA 存储(参数 ssl-cacert) 来启用 mTLS。
存储中没有有效 CA 的设备将在 TLS 握手中断开连接。

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
    ExecStart=/usr/local/bin/rttys run -c /etc/rttys/rttys.conf
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
