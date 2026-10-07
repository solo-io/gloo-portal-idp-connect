FROM golang:1.24 AS builder

WORKDIR /root

COPY . .

RUN make vendor

RUN mkdir -p out

# Static binary (no cgo) so the distroless runtime below needs no libc.
RUN CGO_ENABLED=0 go build -o out/idp-connect ./cmd

# Distroless runtime: ships CA certificates and tzdata only, no shell or package manager, and runs as
# the nonroot user (uid 65532).
FROM cgr.dev/chainguard/static:latest

COPY --from=builder /root/out/idp-connect /app/idp-connect

ENTRYPOINT ["/app/idp-connect"]
