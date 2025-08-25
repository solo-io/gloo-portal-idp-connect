FROM golang:1.24 as builder

WORKDIR /root

COPY . .

RUN make vendor

RUN mkdir -p out

RUN go build -o out/idp-connect ./cmd

FROM ubuntu:24.04

COPY --from=builder /root/out/idp-connect /app/idp-connect

RUN apt-get update && \
  apt-get install --no-install-recommends -y \
  ca-certificates

ENTRYPOINT ["/app/idp-connect"]