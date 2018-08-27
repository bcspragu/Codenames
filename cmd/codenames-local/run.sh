#!/bin/bash
cd ../boardgen-cli
CC=gcc vgo build
cd ../codenames-local
CC=gcc vgo build
./codenames-local --model_file=../../data/everything.bin --words="$(../boardgen-cli/boardgen-cli)"

