#!/bin/bash
set -e

sudo rm -rf /usr/local/go && \
wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
tar -xvf go${GO_VERSION}.linux-amd64.tar.gz && sudo mv go /usr/local/go && \
rm -rf go${GO_VERSION}.linux-amd64.tar.gz

export GOROOT=/usr/local/go && \
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH