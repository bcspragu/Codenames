#!/bin/bash
cd ../boardgen-cli
go build -o boardgen-cli
cd ../codenames-local
go build -o codenames-local
./codenames-local --model_file=../../data/everything.bin --words="$(../boardgen-cli/boardgen-cli)" --use_ai=true
