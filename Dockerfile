FROM golang:1.21 as builder

WORKDIR /root

COPY . .

RUN make vendor

RUN mkdir -p out

RUN go build -o out/idp-connect ./cmd

FROM ubuntu:24.04

COPY --from=builder /root/out/idp-connect /app/idp-connect

ENTRYPOINT ["/app/idp-connect"]