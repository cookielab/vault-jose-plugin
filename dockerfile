FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags "-s -w" -o /out/jose-plugin .

FROM hashicorp/vault:2.0.4

USER root
RUN mkdir -p /vault/plugins
COPY --from=builder /out/jose-plugin /vault/plugins/jose-plugin
RUN chown -R vault:vault /vault/plugins
USER vault

ENV VAULT_ADDR=http://127.0.0.1:8200

EXPOSE 8200

CMD ["server", "-dev", "-dev-plugin-dir=/vault/plugins", "-dev-listen-address=0.0.0.0:8200"]
