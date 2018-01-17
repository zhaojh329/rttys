#!/bin/sh

update-rc.d rttys remove

cp ../../../../bin/rttys /usr/sbin/
cp rttys.init /etc/init.d/rttys
update-rc.d rttys defaults
