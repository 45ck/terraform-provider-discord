package muxserver

import (
	"context"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

// New returns a protocol 6 provider server function that muxes the Framework data sources
// with the SDK resources. This enables incremental migration to terraform-plugin-framework
// without a flag-day rewrite.
func New(ctx context.Context, version string) (func() tfprotov6.ProviderServer, error) {
	upgradedSDK, err := tf5to6server.UpgradeServer(ctx, discord.Provider().GRPCProvider)
	if err != nil {
		return nil, err
	}

	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(fw.New(version)()),
		func() tfprotov6.ProviderServer { return upgradedSDK },
	}

	mux, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return nil, err
	}

	return mux.ProviderServer, nil
}
