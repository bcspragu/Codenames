#!/bin/bash
go run github.com/bcspragu/Codenames/cmd/codenames-local \
  --model_file="../../data/everything.bin" \
  --words="$(go run github.com/bcspragu/Codenames/cmd/boardgen-cli)" \
  --use_ai=true
