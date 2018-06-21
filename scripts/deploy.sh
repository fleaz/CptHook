#! /usr/bin/env bash
set -eu

docker build -f Dockerfile -t $IMAGE .
docker login -u $DOCKER_USER -p $DOCKER_PASS
docker push $IMAGE
