#!/bin/bash
go run github.com/bcspragu/Codenames/cmd/codenames-client \
  --team_to_join="$1" \
  --role_to_join="$2"

