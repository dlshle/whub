#!/bin/sh
git restore .
git pull
vim ./main.go
go build
