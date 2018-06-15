# rttys

[1]: https://img.shields.io/badge/license-LGPL2-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-2.0.2-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]

[rtty](https://github.com/zhaojh329/rtty)的服务端。

`请保持关注以获取最新的项目动态`

# How to install
## 根据自己的平台下载编译好的文件

https://github.com/zhaojh329/rttys/releases

## 解压到你的根目录

	sudo tar -zxvf rttys-x64.tar.gz -C /

## 手动运行

    rttys -cert /etc/rttys/rttys.crt -key /etc/rttys/rttys.key

## 查看支持哪些命令参数

	$ rttys -h
	Usage of rttys:
	  -cert string
	        certFile Path
	  -key string
	        keyFile Path
	  -port int
	        http service port (default 5912)

## 后台运行 (Ubuntu)

	sudo update-rc.d rttys defaults
    sudo /etc/init.d/rttys start

# 贡献代码
如果你想帮助[rttys](https://github.com/zhaojh329/rttys)变得更好，请参考
[CONTRIBUTING_ZH.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING_ZH.md)。

# 技术交流
QQ群：153530783
