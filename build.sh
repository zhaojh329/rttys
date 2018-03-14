#!/bin/sh

cp -r root root_tmp
mkdir -p root_tmp/usr/local/bin

export GOARCH="amd64"
go build
mv rttys root_tmp/usr/local/bin
tar zcvf rttys-x64.tar.gz -C root_tmp/ etc usr


export GOARCH="386"
go build
mv rttys root_tmp/usr/local/bin
tar zcvf rttys.tar.gz -C root_tmp/ etc usr
rm root_tmp -r
