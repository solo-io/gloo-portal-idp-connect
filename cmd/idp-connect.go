package main

import (
	"context"
	"log"

	"github.com/spf13/cobra"

	"github.com/solo-io/gloo-portal-idp-connect/internal/cognito"
	"github.com/solo-io/gloo-portal-idp-connect/internal/version"
)

func main() {
	ctx := context.Background()

	if err := rootCommand(ctx).Execute(); err != nil {
		log.Fatal(err)
	}
}

func rootCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Short:   "IDP Connect sample implementations",
		Version: version.Version,
		// option to silence usage when an error occurs.
		SilenceUsage: true,
	}

	cmd.AddCommand(
		cognito.Command(),
	)

	return cmd
}
