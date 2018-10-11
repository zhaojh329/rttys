#!/bin/sh

targets=linux/386,linux/amd64,linux/arm,linux/arm64,linux/mips,linux/mips64,linux/mipsle,linux/mips64le,windows/*,darwin/*

xgo -targets=$targets -ldflags="-s -w" -dest=bin .

sudo chown -R `id -un` bin

cd bin

targets=$(ls)
for t in $targets
do
	mv $t rttys
	mkdir $t
	mv rttys $t
	cp ../conf/rttys.crt $t
	cp ../conf/rttys.key $t

	echo $t | grep "windows" > /dev/null
	if [ $? -eq 0 ];
	then
		t=$(echo -n $t | sed 's/.exe//')
		mv $t.exe $t
		mv $t/rttys $t/rttys.exe
		cp ../conf/rttys.ini $t
	else
		tar zcvf $t.tar.gz $t --remove-files
	fi
done

cd -
