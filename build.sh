#!/bin/sh

generate() {
	local os="$1"
	local arch="$2"
	local dir="rttys-$os-$arch"
	local bin="rttys"

	mkdir -p output/$dir
	cp conf/rttys.crt conf/rttys.key output/$dir

	[ "$os" = "windows" ] && {
		cp conf/rttys.ini output/$dir
		bin="rttys.exe"
	}

	GOOS=$os GOARCH=$arch go build -ldflags='-s -w' -o output/$dir/$bin

	cd output

	if [ "$os" = "windows" ];
	then
		zip -r $dir.zip $dir
		rm -r $dir
	else
		tar zcvf $dir.tar.gz $dir --remove-files
	fi

	cd ..
}

rm -rf output

generate linux amd64
generate linux 386

generate windows amd64
generate windows 386
