package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ provider.Provider = &PeakhourProvider{}

type PeakhourProvider struct {
	version string
}

type PeakhourProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func (p *PeakhourProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "peakhour"
	resp.Version = p.version
}

func (p *PeakhourProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Peakhour CDN and edge security platform.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "Peakhour API key for authentication. May also be provided via PEAKHOUR_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Peakhour API base URL. Defaults to https://console.peakhour.io. May also be provided via PEAKHOUR_BASE_URL environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *PeakhourProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config PeakhourProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get API key from config or environment
	apiKey := os.Getenv("PEAKHOUR_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing API Key",
			"The provider cannot create the Peakhour API client as there is a missing or empty value for the API key. "+
				"Set the api_key value in the configuration or use the PEAKHOUR_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		return
	}

	// Get base URL from config or environment
	baseURL := os.Getenv("PEAKHOUR_BASE_URL")
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	// Create API client
	c := client.NewClient(apiKey, baseURL)
	if p.version != "" {
		c.UserAgent = fmt.Sprintf("terraform-provider-peakhour/%s", p.version)
	} else {
		c.UserAgent = "terraform-provider-peakhour"
	}
	c.Headers = map[string]string{
		"X-Peakhour-Client": c.UserAgent,
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *PeakhourProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewDomainPlanResource,
		NewReverseProxyServiceResource,
		NewReverseProxyConfigResource,
		NewRPSettingsResource,
		NewRPSSLConfigResource,
		NewRPSSLCertificateResource,
		NewAcmeSettingsResource,
		NewAcmeCertificateResource,
		NewOriginPoolResource,
		NewRPOriginConfigResource,
		NewRPCDNCacheResource,
		NewRPCDNPurgeResourcesResource,
		NewRPCDNPurgeWildcardResource,
		NewRPCDNPurgeTagsResource,
		NewRPBotsResource,
		NewRPThreatAccessListRuleResource,
		NewRPThreatBlockListResource,
		NewRPFirewallSettingsResource,
		NewRPFirewallErrorPageResource,
		NewRPLuaOptionsResource,
		NewRPWAFOptionsResource,
		NewRPWAFOWASPSettingsResource,
		NewRPWAFCustomRuleResource,
		NewRPWAFCustomRuleOrderResource,
		NewRPWAFRuleGroupResource,
		NewTransformSettingsResource,
		NewRuleResource,
		NewRulePhaseOrderResource,
		NewBulkRedirectListResource,
		NewBulkRedirectEntryResource,
		NewRateLimitSettingsResource,
		NewRateLimitGlobalResource,
		NewRateLimitZoneResource,
		NewRuleListResource,
		NewImageTransformResource,
		NewImageTransformCommitResource,
	}
}

func (p *PeakhourProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PeakhourProvider{
			version: version,
		}
	}
}
