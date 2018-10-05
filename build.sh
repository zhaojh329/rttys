#!/bin/sh

targets=linux/386,linux/amd64,linux/arm,linux/arm64,linux/mips,linux/mips64,linux/mipsle,linux/mips64le,windows/*,darwin/*

xgo --targets=$targets .
