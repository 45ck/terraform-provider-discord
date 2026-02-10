package fw

import (
	"context"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// New returns a constructor for the Framework side of the provider.
// This provider is used with terraform-plugin-mux during migration.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &discordProvider{version: version}
	}
}

type discordProvider struct {
	version string
}

type providerModel struct {
	Token    types.String `tfsdk:"token"`
	ClientID types.String `tfsdk:"client_id"`
	Secret   types.String `tfsdk:"secret"`
}

func (p *discordProvider) Metadata(_ context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	// Provider metadata request does not include the type name; set it explicitly.
	resp.TypeName = "discord"
	resp.Version = p.version
}

func (p *discordProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"client_id": schema.StringAttribute{
				Optional: true,
			},
			"secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *discordProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := &discord.Config{
		Token:    cfg.Token.ValueString(),
		ClientID: cfg.ClientID.ValueString(),
		Secret:   cfg.Secret.ValueString(),
	}

	client, err := c.Client()
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}

	// ProviderData is passed into DataSource/Resource Configure.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *discordProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGuildSettingsResource,
	}
}

func (p *discordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewColorDataSource,
		NewLocalImageDataSource,
		NewPermissionDataSource,
		NewServerDataSource,
		NewRoleDataSource,
		NewMemberDataSource,
		NewSystemChannelDataSource,
		NewChannelDataSource,
		NewAPIRequestDataSource,
		NewThreadMembersDataSource,
		NewEmojisDataSource,
		NewStickersDataSource,
		NewSoundboardSoundsDataSource,
		NewSoundboardDefaultSoundsDataSource,
	}
}

func getContextFromProviderData(d any) (*discord.Context, diag.Diagnostics) {
	if d == nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Provider not configured", "provider data was nil")}
	}
	c, ok := d.(*discord.Context)
	if !ok || c == nil || c.Rest == nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Provider not configured", "provider data was not a valid Discord context")}
	}
	return c, nil
}
