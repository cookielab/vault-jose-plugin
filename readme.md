# Vault JWT Secrets Plugin

A HashiCorp Vault secrets plugin that signs and validates JWTs. Based on the
proposal in https://github.com/hashicorp/vault/issues/1986, supporting JWT only
(not JWE or JWS as the proposal outlines).

This is a fork of [naveego/vault-jose-plugin](https://github.com/naveego/vault-jose-plugin),
which stopped receiving changes in 2020. It exists to keep the plugin loadable
by current Vault releases.

## Why fork

The upstream plugin pins `hashicorp/vault/sdk` v0.1.13 (2019) and
`gopkg.in/square/go-jose.v2`, which is archived and carries a decompression bomb
advisory (GHSA-c5q2-7r4c-mv6g). It also depends on a pinned version of the SAP
HANA driver that no longer resolves, so the module cannot be built at all as it
stands. This fork moves to a supported dependency set.

## Requirements

- Go 1.25.7 or newer, required by `hashicorp/vault/sdk`

## Building

```sh
./build.sh
```

Produces `build/jose-plugin` and `build/jose-plugin.sha`. Set `GOOS` and `GOARCH`
to cross-compile.

Without a local Go toolchain:

```sh
docker run --rm -v "$PWD":/src -w /src golang:1.25 \
  sh -c 'CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o build/jose-plugin .'
```

## Vault compatibility

Verified against Vault 2.0.4. The plugin speaks plugin protocol v5, so it should
load on any Vault release that still supports v5.

## Installing

Place the binary in Vault's `plugin_directory`, then register and mount it:

```sh
vault plugin register \
  -sha256="$(cat jose-plugin.sha)" \
  -command=jose-plugin \
  secret jose-plugin

vault secrets enable -path=jwt-issuer jose-plugin
```

See also https://developer.hashicorp.com/vault/docs/plugins.

## Usage

Create a key set with one key in it. The first path segment names the key set,
the second names the key:

```sh
vault write jwt-issuer/jwks/my-app/my-app alg=RS256 use=sig
```

Create a role. `allowed_custom_claims` lists the claims a caller may supply at
issue time; `claims` are fixed values the role always stamps on the token:

```sh
vault write jwt-issuer/roles/my-app - <<'JSON'
{
  "key_set": "my-app",
  "type": "jwt",
  "token_ttl": 3600,
  "allowed_custom_claims": ["uid", "tid"],
  "claims": {
    "iss": "https://vault.example.com/v1/jwt-issuer/jwks/my-app/my-app",
    "env": "dev"
  }
}
JSON
```

Issue a token:

```sh
vault write -field=token jwt-issuer/jwt/issue/my-app - <<'JSON'
{"claims": {"uid": "u-123", "tid": "t-456"}}
JSON
```

Validate one:

```sh
vault write jwt-issuer/jwt/validate/my-app token="$TOKEN"
# is_valid    true
```

The public keys of a key set are readable at `jwt-issuer/jwks/<key set>`, which
is what the `iss` claim above points at.

## Behaviour differences from upstream

Token parsing now accepts only the signature algorithms declared by the keys in
the role's key set. Upstream accepted whatever algorithm the token itself
declared, which is the shape of an algorithm confusion attack. A key set that
declares no algorithms is rejected rather than falling back to permitting
everything.

## Tools

- `./build.sh` builds the plugin and writes its SHA-256 next to it
- `./docker.sh` builds a Vault image with the plugin already registered and runs
  it in dev mode on port 8200 with a root token of `root`
- `./smoke/smoke.sh` builds, installs into Vault and exercises signing

Note that the dev image puts only the binary in the plugin directory. Vault's
`-dev-plugin-dir` tries to register every file it finds there, so a stray `.sha`
alongside it aborts startup.
