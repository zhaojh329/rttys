#!/bin/sh

update-rc.d rttys remove

which rttys > /dev/null && rm `which rttys`
cp ../../../../bin/rttys /usr/local/sbin/
cp rttys.init /etc/init.d/rttys
update-rc.d rttys defaults
