#!/bin/bash
docker run --rm -it -p 8081:8081 -v $PWD:/project node-env yarn serve
