#!/bin/sh
for file in *_test.c
do
    gcc -o "$file.o" "$file"
    ./"$file.o"
done
