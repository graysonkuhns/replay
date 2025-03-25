#!/usr/bin/env bash

rm -rf docs
mkdir -p docs
go build
./replay generateDocs
