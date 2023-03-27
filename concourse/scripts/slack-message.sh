#!/bin/sh
set -eou pipefail

cd ./fauna-go-repository

PACKAGE_VERSION=$(cat version)

echo "fauna-go@${PACKAGE_VERSION} has been released" > ../slack-message/publish

