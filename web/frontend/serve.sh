#!/bin/bash
docker run --rm -it \
  -w /project \
  --net=host \
  --volume $PWD:/project \
  --user $(id -u):$(id -g) \
  node:alpine yarn serve
