package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &ReverseProxyConfigResource{}
var _ resource.ResourceWithConfigure = &ReverseProxyConfigResource{}
var _ resource.ResourceWithImportState = &ReverseProxyConfigResource{}

func NewReverseProxyConfigResource() resource.Resource {
	return &ReverseProxyConfigResource{}
}

type ReverseProxyConfigResource struct {
	client *client.Client
}

type ReverseProxyConfigResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	Websocket          types.Bool   `tfsdk:"websocket"`
	Gzip               types.Bool   `tfsdk:"gzip"`
	Brotli             types.Bool   `tfsdk:"brotli"`
	Aliases            types.List   `tfsdk:"aliases"`
	TrackSessions      types.Bool   `tfsdk:"track_sessions"`
	Debug              types.Bool   `tfsdk:"debug"`
	Segment            types.Bool   `tfsdk:"segment"`
	RedirectMode       types.String `tfsdk:"redirect_mode"`
	RedirectLocation   types.String `tfsdk:"redirect_location"`
	RedirectStatusCode types.Int64  `tfsdk:"redirect_status_code"`
}

func (r *ReverseProxyConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_proxy_config"
}

func (r *ReverseProxyConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Reverse Proxy configuration for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Configuration identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "Domain name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"websocket": schema.BoolAttribute{
				Description: "Enable WebSocket support.",
				Optional:    true,
				Computed:    true,
			},
			"gzip": schema.BoolAttribute{
				Description: "Enable gzip compression.",
				Optional:    true,
				Computed:    true,
			},
			"brotli": schema.BoolAttribute{
				Description: "Enable Brotli compression.",
				Optional:    true,
				Computed:    true,
			},
			"aliases": schema.ListAttribute{
				Description: "Domain aliases.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"track_sessions": schema.BoolAttribute{
				Description: "Enable session tracking.",
				Optional:    true,
				Computed:    true,
			},
			"debug": schema.BoolAttribute{
				Description: "Enable debug mode.",
				Optional:    true,
				Computed:    true,
			},
			"segment": schema.BoolAttribute{
				Description: "Enable segment analytics.",
				Optional:    true,
				Computed:    true,
			},
			"redirect_mode": schema.StringAttribute{
				Description: "Redirect mode (e.g., 'all', 'http').",
				Optional:    true,
			},
			"redirect_location": schema.StringAttribute{
				Description: "Redirect target URL.",
				Optional:    true,
			},
			"redirect_status_code": schema.Int64Attribute{
				Description: "HTTP status code for redirects (301, 302, 307, 308).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *ReverseProxyConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *ReverseProxyConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReverseProxyConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build config update
	config := r.buildConfigFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update config via API
	err := r.client.UpdateReverseProxyConfig(plan.Domain.ValueString(), config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating reverse proxy configuration",
			"Could not create config for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set ID
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/config")

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ReverseProxyConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ReverseProxyConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get config from API
	config, err := r.client.GetReverseProxyConfig(state.Domain.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading reverse proxy configuration",
			"Could not read config for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	r.mapConfigToModel(ctx, config, &state)
	state.ID = types.StringValue(state.Domain.ValueString() + "/config")

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ReverseProxyConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ReverseProxyConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build config update
	config := r.buildConfigFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update config via API
	err := r.client.UpdateReverseProxyConfig(plan.Domain.ValueString(), config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating reverse proxy configuration",
			"Could not update config for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ReverseProxyConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ReverseProxyConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.deleteConfig(ctx, state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error resetting reverse proxy config", err.Error())
		return
	}
}

func (r *ReverseProxyConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *ReverseProxyConfigResource) deleteConfig(ctx context.Context, domain string) error {
	// Send config with explicit zero values to reset all fields
	f := false
	s := ""
	i := 0
	emptySlice := []string{}

	config := client.ReverseProxyConfig{
		Websocket:          &f,
		Gzip:               &f,
		Brotli:             &f,
		Aliases:            &emptySlice,
		TrackSessions:      &f,
		Debug:              &f,
		Segment:            &f,
		RedirectMode:       &s,
		RedirectLocation:   &s,
		RedirectStatusCode: &i,
	}

	err := r.client.UpdateReverseProxyConfig(domain, config)
	return err
}

func (r *ReverseProxyConfigResource) mapConfigToModel(ctx context.Context, config *client.ReverseProxyConfig, state *ReverseProxyConfigResourceModel) {
	if config.Websocket != nil {
		state.Websocket = types.BoolValue(*config.Websocket)
	} else {
		state.Websocket = types.BoolNull()
	}
	if config.Gzip != nil {
		state.Gzip = types.BoolValue(*config.Gzip)
	} else {
		state.Gzip = types.BoolNull()
	}
	if config.Brotli != nil {
		state.Brotli = types.BoolValue(*config.Brotli)
	} else {
		state.Brotli = types.BoolNull()
	}
	if config.TrackSessions != nil {
		state.TrackSessions = types.BoolValue(*config.TrackSessions)
	} else {
		state.TrackSessions = types.BoolNull()
	}
	if config.Debug != nil {
		state.Debug = types.BoolValue(*config.Debug)
	} else {
		state.Debug = types.BoolNull()
	}
	if config.Segment != nil {
		state.Segment = types.BoolValue(*config.Segment)
	} else {
		state.Segment = types.BoolNull()
	}
	if config.RedirectMode != nil {
		state.RedirectMode = types.StringValue(*config.RedirectMode)
	} else {
		state.RedirectMode = types.StringNull()
	}
	if config.RedirectLocation != nil {
		state.RedirectLocation = types.StringValue(*config.RedirectLocation)
	} else {
		state.RedirectLocation = types.StringNull()
	}
	if config.RedirectStatusCode != nil {
		state.RedirectStatusCode = types.Int64Value(int64(*config.RedirectStatusCode))
	} else {
		state.RedirectStatusCode = types.Int64Null()
	}

	if config.Aliases != nil && len(*config.Aliases) > 0 {
		aliases := make([]attr.Value, len(*config.Aliases))
		for i, alias := range *config.Aliases {
			aliases[i] = types.StringValue(alias)
		}
		state.Aliases = types.ListValueMust(types.StringType, aliases)
	} else {
		state.Aliases = types.ListNull(types.StringType)
	}
}

func (r *ReverseProxyConfigResource) buildConfigFromModel(ctx context.Context, model *ReverseProxyConfigResourceModel, diags *diag.Diagnostics) client.ReverseProxyConfig {
	config := client.ReverseProxyConfig{}

	if !model.Websocket.IsNull() {
		v := model.Websocket.ValueBool()
		config.Websocket = &v
	}
	if !model.Gzip.IsNull() {
		v := model.Gzip.ValueBool()
		config.Gzip = &v
	}
	if !model.Brotli.IsNull() {
		v := model.Brotli.ValueBool()
		config.Brotli = &v
	}
	if !model.TrackSessions.IsNull() {
		v := model.TrackSessions.ValueBool()
		config.TrackSessions = &v
	}
	if !model.Debug.IsNull() {
		v := model.Debug.ValueBool()
		config.Debug = &v
	}
	if !model.Segment.IsNull() {
		v := model.Segment.ValueBool()
		config.Segment = &v
	}
	if !model.RedirectMode.IsNull() {
		v := model.RedirectMode.ValueString()
		config.RedirectMode = &v
	}
	if !model.RedirectLocation.IsNull() {
		v := model.RedirectLocation.ValueString()
		config.RedirectLocation = &v
	}
	if !model.RedirectStatusCode.IsNull() {
		v := int(model.RedirectStatusCode.ValueInt64())
		config.RedirectStatusCode = &v
	}

	if !model.Aliases.IsNull() {
		var aliases []string
		d := model.Aliases.ElementsAs(ctx, &aliases, false)
		diags.Append(d...)
		if !diags.HasError() {
			config.Aliases = &aliases
		}
	}

	return config
}
