#!/bin/bash
if [ "$(uname -s)" == "Linux" ]; then
    echo "detected linux, installing build-essential for c compiler"
    sudo apt-get install build-essential -y
fi
gcc -o encode simple_cache_encoder.c
gcc -o decode simple_cache_decoder.c
chmod +x encode decode
