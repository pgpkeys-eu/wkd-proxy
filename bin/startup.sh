#!/bin/sh

path=/wkd-proxy
bin=$path/bin
config=$path/etc/wkd-proxy.conf

if [ ! -f $config ]
then
  cat << EOF >&2
$config missing!
ABORTING
EOF
  exit 1
fi

exec $bin/wkd-proxy -config $config
