#!/bin/bash
docker run --rm -it -v $PWD:/project node-env yarn build
