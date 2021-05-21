#!/bin/bash
go build github.com/bcspragu/Codenames/cmd/codenames-client
./codenames-client --game_to_join="$2" --team_to_join="$3" --role_to_join="$4" "$1"
