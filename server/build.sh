#!/bin/bash
rm -r /Users/zero/Documents/go/src/dog-tunnel/server/DogServer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /Users/zero/Documents/go/src/dog-tunnel/server/DogServer /Users/zero/Documents/go/src/dog-tunnel/main.go
