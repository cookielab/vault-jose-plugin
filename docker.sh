#!/bin/sh

set -e

docker build --file dockerfile --tag vault-jose .
docker run --rm --name vault-jose \
  --publish 8200:8200 \
  --cap-add IPC_LOCK \
  --env VAULT_DEV_ROOT_TOKEN_ID=root \
  vault-jose
