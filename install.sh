#!/bin/sh

update-rc.d rttys remove

which rttys > /dev/null && rm `which rttys`
cp ../../../../bin/rttys /usr/local/sbin/
cp rttys.init /etc/init.d/rttys

rm -rf /etc/rttys

mkdir /etc/rttys
cp rtty.crt /etc/rttys
cp rtty.key /etc/rttys

update-rc.d rttys defaults
