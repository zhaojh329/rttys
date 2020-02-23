#!/bin/sh

VersionPath="github.com/zhaojh329/rttys/version"
GitCommit=$(git log --pretty=format:"%h" -1)
BuildTime=$(date +%FT%T%z)

generate() {
	local os="$1"
	local arch="$2"
	local dir="rttys-$os-$arch"
	local bin="rttys"

	mkdir output/$dir
	cp rttys.conf output/$dir

	[ "$os" = "windows" ] && {
		bin="rttys.exe"
	}

	GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags="-s -w -X $VersionPath.gitCommit=$GitCommit -X $VersionPath.buildTime=$BuildTime" -o output/$dir/$bin

	cd output

	if [ "$os" = "windows" ];
	then
		zip -r $dir.zip $dir
	else
		tar zcvf $dir.tar.gz $dir
	fi

	cd ..
}

rm -rf output
mkdir output

generate linux amd64
generate linux arm64
generate darwin amd64
generate freebsd amd64
generate windows amd64
