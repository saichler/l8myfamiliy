#!/usr/bin/env bash
set -e
cp ~/.netrc .
docker build --no-cache --platform=linux/amd64 -t saichler/family-web:latest .
rm .netrc
docker push saichler/family-web:latest
