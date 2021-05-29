#!/bin/bash
go build github.com/bcspragu/Codenames/cmd/codenames-client
./codenames-client --game_to_join="$2" "$1"
