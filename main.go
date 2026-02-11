package main

import (
	"context"
	"log"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/45ck/terraform-provider-discord/internal/fw"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	ctx := context.Background()

	discord.SetBuildVersion(version)

	// Address can be any stable string; it is used for Terraform CLI dev overrides and debugging.
	const address = "registry.terraform.io/Chaotic-Logic/discord"
	if err := providerserver.Serve(ctx, fw.New(version), providerserver.ServeOpts{Address: address}); err != nil {
		log.Fatalf("failed to serve provider: %v", err)
	}
}
