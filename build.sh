#!/bin/bash
gcc -o encode simple_cache_encoder.c
gcc -o decode simple_cache_decoder.c
chmod +x encode decode
