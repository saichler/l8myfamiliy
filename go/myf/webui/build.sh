#!/usr/bin/env bash
set -e
cp ~/.netrc .
docker build --no-cache --platform=linux/amd64 -t registry.layer-8.dev:44443/my-family/family-web:latest .
rm .netrc
docker push registry.layer-8.dev:44443/my-family/family-web:latest
