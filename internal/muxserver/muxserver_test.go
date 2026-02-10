package muxserver

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
)

func TestMuxServer_New_DoesNotError(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), "test")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
}

func TestProviderSchemaParity_SDKVsFramework(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	upgradedSDK, err := tf5to6server.UpgradeServer(ctx, discord.Provider().GRPCProvider)
	if err != nil {
		t.Fatalf("UpgradeServer returned error: %v", err)
	}

	fwServer := providerserver.NewProtocol6(fw.New("test")())

	req := &tfprotov6.GetProviderSchemaRequest{}

	sdkSchema, err := upgradedSDK.GetProviderSchema(ctx, req)
	if err != nil {
		t.Fatalf("SDK GetProviderSchema returned error: %v", err)
	}
	fwSchema, err := fwServer().GetProviderSchema(ctx, req)
	if err != nil {
		t.Fatalf("Framework GetProviderSchema returned error: %v", err)
	}

	// With mux, the provider configuration schema MUST match across both servers.
	sdkJSON, _ := json.Marshal(sdkSchema.Provider)
	fwJSON, _ := json.Marshal(fwSchema.Provider)
	if !bytes.Equal(sdkJSON, fwJSON) {
		t.Fatalf("provider configuration schema mismatch\nsdk=%s\nfw=%s", string(sdkJSON), string(fwJSON))
	}
}
