#!/bin/sh

ARCH=$(uname -m)
filename=rttys.tar.gz

[ "$ARCH" = "x86_64" ] && filename=rttys-x64.tar.gz

URL=https://github.com/zhaojh329/rttys/blob/$filename

curl -o rttys.tar.gz -L -f $URL

if [ $? -eq 0 ]
then 
    # unpack:
    tar -zxvf rttys.tar.gz -C /
    if [ $? -eq 0 ]
    then
        rm rttys.tar.gz
        update-rc.d rttys defaults
        echo "rttys has been installed"
        exit 0
    fi
else
    echo "Failed to determine your platform.\nTry compile yourself"
fi

exit 1
