#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

docker run --rm -it \
  -w /project \
  -p 8081:8081 \
  --mount type=bind,source=$DIR,destination=/project \
  node:alpine yarn serve
