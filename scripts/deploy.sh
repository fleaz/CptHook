#! /usr/bin/env bash
set -eu

TAG=$1

docker tag $IMAGE $IMAGE:$TAG
docker login -u $DOCKER_USER -p $DOCKER_PASS
docker push $IMAGE:$TAG
