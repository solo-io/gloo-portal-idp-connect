package main

import (
	"context"
	"flag"
	"log"

	"github.com/solo-io/gloo-portal-idp-connect/cognito/internal/server"
)

func main() {
	serverOpts := &server.Options{}
	serverOpts.AddToFlags(flag.CommandLine)
	flag.Parse()

	ctx := context.Background()

	log.Fatal(server.ListenAndServe(ctx, serverOpts))
}
