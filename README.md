# rttys([中文](/README_ZH.md))

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-4.2.0-blue.svg?style=plastic
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

### Use your own authentication server
If the device authentication URL is configured, when the device connecting,
rttys will initiate an authentication request to this URL, and the authentication
server will return whether the authentication has been passed.

Request data format:

    {"devid":"test", "token":"34762d07637276694b938d23f10d7164"}

Authentication Server Response Format:

    {"auth": true}

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

## nginx proxy

```
# rttys.conf

addr-user: 127.0.0.1:5913

addr-http-proxy: 127.0.0.1:5914
http-proxy-redir-url: http://web.your-server.com
http-proxy-redir-domain: .your-server.com
```

```
# nginx.conf

server {
    listen 80;

    server_name rtty.your-server.com;

    location /connect/ {
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_pass http://127.0.0.1:5913;
    }

    location / {
        proxy_pass http://127.0.0.1:5913;
    }
}

server {
    listen 80;

    server_name web.your-server.com;

    location / {
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_pass http://127.0.0.1:5914;
    }
}
```

## Docker
### Simple run

    sudo docker run -it -p 5912:5912 -p 5913:5913 -p 5914:5914 zhaojh329/rttys:latest \
        run --addr-http-proxy :5914

### Using config file

    sudo mkdir -p /opt/rttys
    sudo sh -c 'echo "addr-http-proxy: :5914" > /opt/rttys/rttys.conf'
    sudo docker run -it -p 5912:5912 -p 5913:5913 -p 5914:5914 -v /opt/rttys:/etc/rttys \
        zhaojh329/rttys:latest run -conf /etc/rttys/rttys.conf

## Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.
