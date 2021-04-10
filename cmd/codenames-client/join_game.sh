#!/bin/bash
go run github.com/bcspragu/Codenames/cmd/codenames-client \
  --game_to_join="$1" \
  --team_to_join="$2" \
  --role_to_join="$3"


