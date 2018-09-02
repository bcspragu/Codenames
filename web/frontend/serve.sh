#!/bin/bash
docker run --rm -it --net=host -v $PWD:/project node-env yarn serve
