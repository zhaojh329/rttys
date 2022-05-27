# rttys

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-4.0.1-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://github.com/zhaojh329/rttys/workflows/build/badge.svg

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
![Build Status][9]

这是[rtty](https://github.com/zhaojh329/rtty)的服务器程序。

## 认证(可选)
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

## 数据库准备
### Sqlite
sqlite://rttys.db

### MySql 或者 Mariadb
mysql://rttys:rttys@tcp(localhost)/rttys

在数据库实例上，以root用户身份登录到数据库控制台：
```
mysql -u root -p
```

创建将由 Rttys 使用的数据库用户，通过密码验证。本例使用 “rttys” 作为密码。请为您的实例使用安全密码。
```
CREATE USER 'rttys' IDENTIFIED BY 'rttys';
```

使用 UTF-8 字符集和排序规则创建数据库。确保使用 utf8mb4 字符集而不是 utf8，因为前者支持基本多语言平面
之外的所有 Unicode字符（包括emojis）。另外，根据您期望的内容选择排序规则。如有疑问，请使用 unicode_ci 或general_ci。
```
CREATE DATABASE rttys CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci';
```
将数据库上的所有权限授予上面创建的数据库用户。
```
GRANT ALL PRIVILEGES ON rttys.* TO 'rttys';
FLUSH PRIVILEGES;
```

退出数据库控制台。

## Docker

    sudo docker run -it -p 5912:5912 -p 5913:5913 zhaojh329/rttys:latest

## 贡献代码
如果你想帮助[rttys](https://github.com/zhaojh329/rttys)变得更好，请参考
[CONTRIBUTING_ZH.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING_ZH.md)。
