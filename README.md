# rttys([中文](/README_ZH.md))

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-4.1.5-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://github.com/zhaojh329/rttys/workflows/build/badge.svg

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
![Build Status][9]

This is the server program of [rtty](https://github.com/zhaojh329/rtty)

## Build from source
golang and node 20+ is required

    cd ui
    npm install
    npm run build
    cd ..
    go build

## Authorization(optional)
### Token
Generate a token

    $ rttys token
    Please set a password:******
    Your token is: 34762d07637276694b938d23f10d7164

Use token

    $ rttys run -t 34762d07637276694b938d23f10d7164

### mTLS
You can enable mTLS by specifying device CA storage (valid file) in config file or from CLI (variable ssl-cacert).
Device(s) without valid CA in storage will be disconnected in TLS handshake.

## Database Preparation
## Sqlite
s#qlite://rttys.db

### MySql or Mariadb
mysql://rttys:rttys@tcp(localhost)/rttys

On database instance, login to database console as root:
```
mysql -u root -p
```

Create database user which will be used by Rttys, authenticated by password. This example uses 'rttys' as password. Please use a secure password for your instance.
```
CREATE USER 'rttys' IDENTIFIED BY 'rttys';
```

Create database with UTF-8 charset and collation. Make sure to use utf8mb4 charset instead of utf8 as the former supports all Unicode characters (including emojis) beyond Basic Multilingual Plane. Also, collation chosen depending on your expected content. When in doubt, use either unicode_ci or general_ci.
```
CREATE DATABASE rttys CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci';
```

Grant all privileges on the database to database user created above.
```
GRANT ALL PRIVILEGES ON rttys.* TO 'rttys';
FLUSH PRIVILEGES;
```

Quit from database console by exit.

## Docker

    sudo docker run -it -p 5912:5912 -p 5913:5913 zhaojh329/rttys:latest

## Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.
