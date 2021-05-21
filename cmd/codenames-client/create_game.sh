#!/bin/bash
go build github.com/bcspragu/Codenames/cmd/codenames-client 
./codenames-client --team_to_join="$2" --role_to_join="$3" "$1"
