#!/usr/bin/env bash

rm -rf docs
mkdir -p docs
cd tools/docgen
go run main.go ../../docs/
