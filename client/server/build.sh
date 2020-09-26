#!/bin/bash
rm -r /Users/zero/Documents/go/src/dog-tunnel/client/server/dogClientServer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /Users/zero/Documents/go/src/dog-tunnel/client/server/dogClientServer /Users/zero/Documents/go/src/dog-tunnel/client/server/main.go
