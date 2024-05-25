#!/bin/sh
for file in *_test.c
do
    gcc -o "$file.o" "$file"
    echo "Running: $file"
    ./"$file.o"
done
