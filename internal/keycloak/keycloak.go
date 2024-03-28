package keycloak

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/solo-io/gloo-portal-idp-connect/internal/keycloak/server"
)

func Command() *cobra.Command {
	serverOpts := &server.Options{}

	cmd := &cobra.Command{
		Short: "Start the Keycloak IDP connector",
		Use:   "keycloak",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.ListenAndServe(context.Background(), serverOpts)
		},
		// option to silence usage when an error occurs
		SilenceUsage: true,
	}

	serverOpts.AddToFlags(cmd.Flags())

	return cmd
}