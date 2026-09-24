# rttys

[1]: https://img.shields.io/badge/license-MIT-brightgreen.svg?style=plastic
[2]: /LICENSE
[3]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=plastic
[4]: https://github.com/zhaojh329/rttys/pulls
[5]: https://img.shields.io/badge/Issues-welcome-brightgreen.svg?style=plastic
[6]: https://github.com/zhaojh329/rttys/issues/new
[7]: https://img.shields.io/badge/release-5.5.2-blue.svg?style=plastic
[8]: https://github.com/zhaojh329/rttys/releases
[9]: https://github.com/zhaojh329/rttys/workflows/build/badge.svg
[10]: https://img.shields.io/github/downloads/zhaojh329/rttys/total
[12]: https://deepwiki.com/badge.svg
[13]: https://deepwiki.com/zhaojh329/rttys
[14]: https://goreportcard.com/badge/github.com/zhaojh329/rttys/v5
[15]: https://goreportcard.com/report/github.com/zhaojh329/rttys/v5

[![license][1]][2]
[![PRs Welcome][3]][4]
[![Issue Welcome][5]][6]
[![Release Version][7]][8]
![Build Status][9]
[![Go Report Card][14]][15]
![Downloads][10]
[![Ask DeepWiki][12]][13]

> **🔗 This is the server component of the rtty project. For complete information please visit the main [rtty client repository](https://github.com/zhaojh329/rtty).**

![](/img/terminal.gif)
![](/img/file.gif)
![](/img/web.gif)
![](/img/virtual-keyboard.jpg)

## 📖 About

**rttys** is the server-side component of the [rtty](https://github.com/zhaojh329/rtty) remote terminal system. It provides a web-based management interface and handles connections from rtty clients running on remote devices.

## 🏗️ Project Structure

This repository contains the server and its browser-based management interface:
- `cmd/rttys/`: CLI entry point and configuration parsing
- `internal/server/`: device connections, API, HTTP proxy, and embedded web assets
- `internal/log/`, `internal/utils/`: server support code
- `ui/`: Vue web interface
- `scripts/`: release and Debian build scripts
- `deploy/`: example configuration and systemd unit

Build the web interface first, then build the server from the repository root:

```sh
cd ui && npm ci && npm run build && cd ..
go build -o rttys ./cmd/rttys
```

Go 1.27 or newer is required. Serial access requires a device client supporting
protocol version 6. On the device list, select the serial icon, choose a detected
port and its baud rate, data bits, stop bits and parity, then open the console in
a new tab. The connection owns the port until the tab is closed.
An existing terminal session continues to work with older clients.

### Temporary shares

Use **Shares** in the device list toolbar to open the management dialog, or use
the share action on a device row to open the creation dialog directly.
Terminal and serial shares expose a temporary SSH endpoint. Connect as `share`
with the password shown once at creation. SSH authentication protects the share;
the device terminal may still show its own login prompt. Only interactive shell
sessions are accepted. Serial shares use the selected port settings and allow one
user per serial port.

TCP shares expose a device-reachable IPv4 address and port as a raw TCP listener.
They have no authentication: anyone who can connect to the share port can reach
that destination. Restrict access at the server firewall as needed. The port is
chosen automatically from `share-port-start` through `share-port-end`, or can be
selected manually within that range. `share-bind-host` controls the listening
interface and `share-public-host` controls the displayed hostname. SSH host keys
are generated under the server account's config directory by default; set
`share-host-key` to use a stable location. Shares close after the configured
period with no successful connections, on device disconnect, or when ended in
the management page. Active connections keep their share open.

Serial and TCP shares require the current protocol v6 client. The new TCP
messages are included in v6; no additional protocol version is introduced.
Both endpoints must include the v6 TCP ACK operation: forwarding uses independent
256 KiB byte windows in each direction so slow receivers pause their own stream.
Earlier development builds of v6 TCP forwarding without ACK support are incompatible.
Browser requests use the `/api/` prefix for all management and device access
endpoints. Page routes such as `/rtty/:devid` are separate from the API.

When `user-hook-url` is configured, share operations send the hook a GET request
with server-generated device and group headers:

```http
X-Rttys-Device-ID: router
X-Rttys-Group: lab
X-Original-Method: POST
```

An empty `X-Rttys-Group` identifies the default group. `X-Original-Method` is
`POST` for creation, `GET` for listing, or `DELETE` for deletion.
Creation uses the parsed, validated request before allocating a listener.
List and delete checks use the stored share's device and group, not values
supplied through query parameters or headers. Each active share in a list is
checked separately, and only entries receiving HTTP 200 are returned; an empty
or fully denied list returns `[]`. Hook failures also deny access.
Creation or deletion denied by the hook returns HTTP 403.

The hook must enforce device-level authorization using `X-Rttys-Device-ID` and
`X-Rttys-Group`. These headers do not provide TCP target or serial port details.
Original identity headers and `X-Original-Method` / `X-Original-URL` remain
available. Incoming `X-Rttys-*` headers are discarded before server-owned hook
headers are set. Other device operations also send these two headers, using
the device and group from their routes or query parameters. With no hook configured, the existing API authentication
applies and shares are not filtered by device.

## ⭐ Star History
[![Star History Chart](https://api.star-history.com/svg?repos=zhaojh329/rttys&type=Date)](https://www.star-history.com/#zhaojh329/rttys&Date)

## 🤝 Contributing
If you would like to help making [rttys](https://github.com/zhaojh329/rttys) better,
see the [CONTRIBUTING.md](https://github.com/zhaojh329/rttys/blob/master/CONTRIBUTING.md) file.

## ❤️ [Donation](https://zhaojh329.github.io/zhaojh329/)
