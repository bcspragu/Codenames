#!/bin/bash
go build github.com/bcspragu/Codenames/cmd/codenames-client 
./codenames-client "$1"
