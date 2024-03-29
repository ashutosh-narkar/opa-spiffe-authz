#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(cd src/privileged && GOOS=linux go build -mod=mod -v -o $DIR/docker/privileged/privileged)
(cd src/restricted && GOOS=linux go build -mod=mod -v -o $DIR/docker/restricted/restricted)
(cd src/external && GOOS=linux go build -mod=mod -v -o $DIR/docker/external/external)
(cd src/db-server && GOOS=linux go build -mod=mod -v -o $DIR/docker/db/db-server)
