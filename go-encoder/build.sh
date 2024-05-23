#!/bin/bash
go build -o encoder .
cp decode.sh decode
cp encode.sh encode
chmod +x encode
chood +x decode
