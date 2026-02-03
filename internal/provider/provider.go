package provider

import (
	"context"
	"os"
	"time"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure Provider satisfies various interfaces.
var _ provider.Provider = &ArcaneProvider{}

// ArcaneProvider defines the provider implementation.
type ArcaneProvider struct {
	version string
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ArcaneProvider{version: version}
	}
}

// Metadata returns the provider type name.
func (p *ArcaneProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "arcane"
	resp.Version = p.version
}

// Schema defines the provider-level configuration schema.
func (p *ArcaneProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Base API endpoint for Arcane (e.g., http://localhost:3552/api).",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key used for authentication (sent as X-API-Key). Can be set via ARCANE_API_KEY.",
				Optional:    true,
				Sensitive:   true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"http_timeout": schema.StringAttribute{
				Description: "HTTP request timeout (e.g., 120s, 2m). Defaults to 120s if unset or invalid.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a configured client for data sources and resources.
func (p *ArcaneProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config struct {
		Endpoint    types.String `tfsdk:"endpoint"`
		APIKey      types.String `tfsdk:"api_key"`
		HTTPTimeout types.String `tfsdk:"http_timeout"`
	}

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := "http://localhost:3552/api"
	if !config.Endpoint.IsNull() && !config.Endpoint.IsUnknown() {
		endpoint = config.Endpoint.ValueString()
	}

	apiKey := os.Getenv("ARCANE_API_KEY")
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() && config.APIKey.ValueString() != "" {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider requires an API key. Set it via the provider attribute api_key or the ARCANE_API_KEY environment variable.",
		)
		return
	}

	// Determine timeout
	timeout := 120 * time.Second
	if !config.HTTPTimeout.IsNull() && !config.HTTPTimeout.IsUnknown() {
		if d, err := time.ParseDuration(config.HTTPTimeout.ValueString()); err == nil && d > 0 {
			timeout = d
		}
	}
	client := sdkclient.NewClientWithTimeout(endpoint, apiKey, timeout)
	tflog.Info(ctx, "Configured Arcane provider", map[string]any{"endpoint": endpoint, "timeout": timeout.String()})

	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources returns the provider data sources (none yet).
func (p *ArcaneProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources returns the provider resources.
func (p *ArcaneProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewSettingsResource,
		NewEnvironmentResource,
		NewProjectResource,
		NewProjectPathResource,
		NewRegistryResource,
		NewNotificationResource,
		NewContainerResource,
		NewGitRepositoryResource,
		NewGitOpsSyncResource,
		NewApiKeyResource,
		NewTemplateResource,
		NewVolumeResource,
		NewNetworkResource,
	}
}
