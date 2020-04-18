#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(cd src/special && GOOS=linux go build -v -o $DIR/docker/special/special)
(cd src/db-server && GOOS=linux go build -v -o $DIR/docker/db/db-server)
