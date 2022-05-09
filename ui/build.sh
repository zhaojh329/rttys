#!/bin/sh

sudo docker run -it \
  -v /etc/group:/etc/group:ro \
  -v /etc/passwd:/etc/passwd:ro \
  -v /etc/shadow:/etc/shadow:ro \
  -v $HOME:$HOME \
  -u $(id -u):$(id -g) \
  -w $(pwd) \
  --rm \
  node:12 \
  npm $*
