# rttys([中文](/README_ZH.md))

[1]: https://img.shields.io/badge/license-LGPL2-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]

The server side of [rtty](https://github.com/zhaojh329/rtty)

`Keep Watching for More Actions on This Space`

# How to install
## Download the compiled file according to your own platform

https://github.com/zhaojh329/rttys/releases

## Decompress the file to your root path

	sudo tar -zxvf rttys-x64.tar.gz -C /

## Manual run

    rttys -cert /etc/rttys/rttys.crt -key /etc/rttys/rttys.key

## See Supported Command Line Parameters

	$ rttys -h
	Usage of rttys:
	  -cert string
	        certFile Path
	  -key string
	        keyFile Path
	  -port int
	        http service port (default 5912)

## Run in background (Ubuntu)

	update-rc.d rttys defaults
    sudo /etc/init.d/rttys start

# Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.
