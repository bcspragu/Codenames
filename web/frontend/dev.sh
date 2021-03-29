#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

docker run --rm -it \
  -w /project \
  --mount type=bind,source=$DIR,destination=/project \
  node:alpine /bin/sh
