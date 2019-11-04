#!/bin/sh

VersionPath="github.com/zhaojh329/rttys/version"
GitCommit=$(git log --pretty=format:"%h" -1)
BuildTime=$(date +%FT%T%z)

generate() {
	local os="$1"
	local arch="$2"
	local dir="rttys-$os-$arch"
	local bin="rttys"
	local cgo=0

	mkdir output/$dir
	cp rttys.conf output/$dir
	cp output/rttys.crt output/rttys.key output/$dir

	[ "$os" = "windows" ] && {
		bin="rttys.exe"
	}

	[ "$os" = "linux" ] && {
		cgo=1
	}

	GOOS=$os GOARCH=$arch CGO_ENABLED=$cgo go build -ldflags="-s -w -X $VersionPath.gitCommit=$GitCommit -X $VersionPath.buildTime=$BuildTime" -o output/$dir/$bin

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

TARGET=output ./generate-CA.sh rttys

generate linux amd64
generate linux 386

generate windows amd64
generate windows 386

generate darwin amd64
generate darwin 386

generate freebsd amd64
generate freebsd 386
