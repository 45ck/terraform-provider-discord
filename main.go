package main

import (
	"context"
	"log"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/muxserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
)

func main() {
	ctx := context.Background()

	discord.SetBuildVersion(version)

	serverFunc, err := muxserver.New(ctx, version)
	if err != nil {
		log.Fatalf("failed to create mux server: %v", err)
	}

	// Address can be any stable string; it is used for Terraform CLI dev overrides and debugging.
	const address = "registry.terraform.io/Chaotic-Logic/discord"
	if err := tf6server.Serve(address, serverFunc); err != nil {
		log.Fatalf("failed to serve provider: %v", err)
	}
}
