#!/bin/sh

cp -r root root_tmp
mkdir -p root_tmp/usr/local/bin

go build
mv rttys root_tmp/usr/local/bin
tar zcvf rttys-x64.tar.gz -C root_tmp/ etc usr
rm root_tmp -r
