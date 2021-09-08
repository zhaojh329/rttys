#!/bin/sh

VersionPath="rttys/version"
GitCommit=$(git log --pretty=format:"%h" -1)
BuildTime=$(date +%FT%T%z)

[ $# -lt 2 ] && {
	echo "Usage: $0 linux amd64"
	exit 1
}

generate() {
	local os="$1"
	local arch="$2"
	local dir="rttys-$os-$arch"
	local bin="rttys"

	mkdir $dir
	cp rttys.conf $dir

	[ "$os" = "windows" ] && {
		bin="rttys.exe"
	}

	GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags="-s -w -X $VersionPath.gitCommit=$GitCommit -X $VersionPath.buildTime=$BuildTime" -o $dir/$bin
}

generate $1 $2
