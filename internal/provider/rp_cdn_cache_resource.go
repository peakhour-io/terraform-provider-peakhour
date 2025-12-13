package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPCDNCacheResource{}
var _ resource.ResourceWithConfigure = &RPCDNCacheResource{}
var _ resource.ResourceWithImportState = &RPCDNCacheResource{}

func NewRPCDNCacheResource() resource.Resource {
	return &RPCDNCacheResource{}
}

type RPCDNCacheResource struct {
	client *client.Client
}

type RPCDNCacheResourceModel struct {
	ID                             types.String `tfsdk:"id"`
	Domain                         types.String `tfsdk:"domain"`
	CacheEnabled                   types.Bool   `tfsdk:"cache_enabled"`
	SoftPurge                      types.Bool   `tfsdk:"soft_purge"`
	CDNQueryMode                   types.String `tfsdk:"cdn_query_mode"`
	BrowserTTLSec                  types.Int64  `tfsdk:"browser_ttl_sec"`
	EdgeTTLSec                     types.Int64  `tfsdk:"edge_ttl_sec"`
	CacheImplicitTTL               types.Int64  `tfsdk:"cache_implicit_ttl"`
	CacheStoreRequireCacheControl  types.Bool   `tfsdk:"cache_store_require_cache_control"`
	CacheStripCookies              types.Bool   `tfsdk:"cache_strip_cookies"`
	CacheStripSetCookies           types.Bool   `tfsdk:"cache_strip_set_cookies"`
	CacheVaryUAMode                types.String `tfsdk:"cache_vary_ua_mode"`
	CDNIgnoreInvalidate            types.Bool   `tfsdk:"cdn_ignore_invalidate"`
	CDNIgnoreVaryUserAgent         types.Bool   `tfsdk:"cdn_ignore_vary_user_agent"`
	CacheIgnoreRequestCacheControl types.Bool   `tfsdk:"cache_ignore_request_cache_control"`
	CDNServeStale                  types.Bool   `tfsdk:"cdn_serve_stale"`
	CDNSkipCookie                  types.String `tfsdk:"cdn_skip_cookie"`
	CacheTagHeader                 types.String `tfsdk:"cache_tag_header"`
	CacheTagHeaderSeparator        types.String `tfsdk:"cache_tag_header_separator"`
	CDNRemoveQueryArgs             types.List   `tfsdk:"cdn_remove_query_args"`
}

func (r *RPCDNCacheResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_cdn_cache"
}

func (r *RPCDNCacheResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	useStateBool := []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}
	useStateInt := []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	useStateStr := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		Description: "Manages RP CDN cache configuration for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP CDN cache config identifier (domain/rp_cdn_cache).",
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
			"cache_enabled": schema.BoolAttribute{
				Description:   "Enable edge caching. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"soft_purge": schema.BoolAttribute{
				Description:   "Enable soft purge. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cdn_query_mode": schema.StringAttribute{
				Description:   "Cache query mode (spec enum: full, none, strip). Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"browser_ttl_sec": schema.Int64Attribute{
				Description:   "Browser TTL in seconds. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"edge_ttl_sec": schema.Int64Attribute{
				Description:   "Edge TTL in seconds. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"cache_implicit_ttl": schema.Int64Attribute{
				Description:   "Implicit cache TTL. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"cache_store_require_cache_control": schema.BoolAttribute{
				Description:   "Only store when Cache-Control allows it. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cache_strip_cookies": schema.BoolAttribute{
				Description:   "Strip Cookie header before caching. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cache_strip_set_cookies": schema.BoolAttribute{
				Description:   "Strip Set-Cookie header before caching. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cache_vary_ua_mode": schema.StringAttribute{
				Description:   "Vary User-Agent mode (spec enum: none, device-type). Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"cdn_ignore_invalidate": schema.BoolAttribute{
				Description:   "Ignore invalidate headers. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cdn_ignore_vary_user_agent": schema.BoolAttribute{
				Description:   "Ignore Vary: User-Agent. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cache_ignore_request_cache_control": schema.BoolAttribute{
				Description:   "Ignore request Cache-Control headers. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cdn_serve_stale": schema.BoolAttribute{
				Description:   "Serve stale content where possible. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"cdn_skip_cookie": schema.StringAttribute{
				Description:   "Cookie name/pattern that disables caching when present. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"cache_tag_header": schema.StringAttribute{
				Description:   "Header name containing cache tags. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"cache_tag_header_separator": schema.StringAttribute{
				Description:   "Cache tag header separator (spec enum: \",\" or \" \"). Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"cdn_remove_query_args": schema.ListAttribute{
				Description: "Query args to remove before caching. Set to null to clear.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPCDNCacheResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPCDNCacheResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPCDNCacheResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyCacheConfig(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP CDN cache config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_cdn_cache")

	if err := r.readCacheConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP CDN cache config after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNCacheResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPCDNCacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readCacheConfig(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP CDN cache config", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_cdn_cache")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPCDNCacheResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPCDNCacheResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyCacheConfig(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP CDN cache config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_cdn_cache")

	if err := r.readCacheConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP CDN cache config after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNCacheResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPCDNCacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"cache_enabled":                      nil,
		"soft_purge":                         nil,
		"cdn_query_mode":                     nil,
		"browser_ttl_sec":                    nil,
		"edge_ttl_sec":                       nil,
		"cache_implicit_ttl":                 nil,
		"cache_store_require_cache_control":  nil,
		"cache_strip_cookies":                nil,
		"cache_strip_set_cookies":            nil,
		"cache_vary_ua_mode":                 nil,
		"cdn_ignore_invalidate":              nil,
		"cdn_ignore_vary_user_agent":         nil,
		"cache_ignore_request_cache_control": nil,
		"cdn_serve_stale":                    nil,
		"cdn_skip_cookie":                    nil,
		"cache_tag_header":                   nil,
		"cache_tag_header_separator":         nil,
		"cdn_remove_query_args":              nil,
	}

	if err := r.client.UpdateRPCDNCacheConfig(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting RP CDN cache config", err.Error())
		return
	}
}

func (r *RPCDNCacheResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPCDNCacheResource) applyCacheConfig(ctx context.Context, model *RPCDNCacheResourceModel, diags *diag.Diagnostics) error {
	update := map[string]any{}

	addBool := func(key string, v types.Bool) {
		if v.IsUnknown() {
			return
		}
		if v.IsNull() {
			update[key] = nil
			return
		}
		update[key] = v.ValueBool()
	}
	addInt := func(key string, v types.Int64) {
		if v.IsUnknown() {
			return
		}
		if v.IsNull() {
			update[key] = nil
			return
		}
		update[key] = int(v.ValueInt64())
	}
	addString := func(key string, v types.String) {
		if v.IsUnknown() {
			return
		}
		if v.IsNull() {
			update[key] = nil
			return
		}
		update[key] = v.ValueString()
	}

	addBool("cache_enabled", model.CacheEnabled)
	addBool("soft_purge", model.SoftPurge)
	addString("cdn_query_mode", model.CDNQueryMode)
	addInt("browser_ttl_sec", model.BrowserTTLSec)
	addInt("edge_ttl_sec", model.EdgeTTLSec)
	addInt("cache_implicit_ttl", model.CacheImplicitTTL)
	addBool("cache_store_require_cache_control", model.CacheStoreRequireCacheControl)
	addBool("cache_strip_cookies", model.CacheStripCookies)
	addBool("cache_strip_set_cookies", model.CacheStripSetCookies)
	addString("cache_vary_ua_mode", model.CacheVaryUAMode)
	addBool("cdn_ignore_invalidate", model.CDNIgnoreInvalidate)
	addBool("cdn_ignore_vary_user_agent", model.CDNIgnoreVaryUserAgent)
	addBool("cache_ignore_request_cache_control", model.CacheIgnoreRequestCacheControl)
	addBool("cdn_serve_stale", model.CDNServeStale)
	addString("cdn_skip_cookie", model.CDNSkipCookie)
	addString("cache_tag_header", model.CacheTagHeader)
	addString("cache_tag_header_separator", model.CacheTagHeaderSeparator)

	if !model.CDNRemoveQueryArgs.IsUnknown() {
		if model.CDNRemoveQueryArgs.IsNull() {
			update["cdn_remove_query_args"] = nil
		} else {
			var args []string
			diags.Append(model.CDNRemoveQueryArgs.ElementsAs(ctx, &args, false)...)
			if diags.HasError() {
				return fmt.Errorf("invalid cdn_remove_query_args list")
			}
			update["cdn_remove_query_args"] = args
		}
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPCDNCacheConfig(model.Domain.ValueString(), update)
}

func (r *RPCDNCacheResource) readCacheConfig(ctx context.Context, state *RPCDNCacheResourceModel) error {
	cfg, err := r.client.GetRPCDNCacheConfig(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if cfg.CacheEnabled != nil {
		state.CacheEnabled = types.BoolValue(*cfg.CacheEnabled)
	} else {
		state.CacheEnabled = types.BoolNull()
	}
	if cfg.SoftPurge != nil {
		state.SoftPurge = types.BoolValue(*cfg.SoftPurge)
	} else {
		state.SoftPurge = types.BoolNull()
	}
	if cfg.CDNQueryMode != nil {
		state.CDNQueryMode = types.StringValue(*cfg.CDNQueryMode)
	} else {
		state.CDNQueryMode = types.StringNull()
	}
	if cfg.BrowserTTLSec != nil {
		state.BrowserTTLSec = types.Int64Value(int64(*cfg.BrowserTTLSec))
	} else {
		state.BrowserTTLSec = types.Int64Null()
	}
	if cfg.EdgeTTLSec != nil {
		state.EdgeTTLSec = types.Int64Value(int64(*cfg.EdgeTTLSec))
	} else {
		state.EdgeTTLSec = types.Int64Null()
	}
	if cfg.CacheImplicitTTL != nil {
		state.CacheImplicitTTL = types.Int64Value(int64(*cfg.CacheImplicitTTL))
	} else {
		state.CacheImplicitTTL = types.Int64Null()
	}
	if cfg.CacheStoreRequireCacheControl != nil {
		state.CacheStoreRequireCacheControl = types.BoolValue(*cfg.CacheStoreRequireCacheControl)
	} else {
		state.CacheStoreRequireCacheControl = types.BoolNull()
	}
	if cfg.CacheStripCookies != nil {
		state.CacheStripCookies = types.BoolValue(*cfg.CacheStripCookies)
	} else {
		state.CacheStripCookies = types.BoolNull()
	}
	if cfg.CacheStripSetCookies != nil {
		state.CacheStripSetCookies = types.BoolValue(*cfg.CacheStripSetCookies)
	} else {
		state.CacheStripSetCookies = types.BoolNull()
	}
	if cfg.CacheVaryUAMode != nil {
		state.CacheVaryUAMode = types.StringValue(*cfg.CacheVaryUAMode)
	} else {
		state.CacheVaryUAMode = types.StringNull()
	}
	if cfg.CDNIgnoreInvalidate != nil {
		state.CDNIgnoreInvalidate = types.BoolValue(*cfg.CDNIgnoreInvalidate)
	} else {
		state.CDNIgnoreInvalidate = types.BoolNull()
	}
	if cfg.CDNIgnoreVaryUserAgent != nil {
		state.CDNIgnoreVaryUserAgent = types.BoolValue(*cfg.CDNIgnoreVaryUserAgent)
	} else {
		state.CDNIgnoreVaryUserAgent = types.BoolNull()
	}
	if cfg.CacheIgnoreRequestCacheControl != nil {
		state.CacheIgnoreRequestCacheControl = types.BoolValue(*cfg.CacheIgnoreRequestCacheControl)
	} else {
		state.CacheIgnoreRequestCacheControl = types.BoolNull()
	}
	if cfg.CDNServeStale != nil {
		state.CDNServeStale = types.BoolValue(*cfg.CDNServeStale)
	} else {
		state.CDNServeStale = types.BoolNull()
	}
	if cfg.CDNSkipCookie != nil {
		state.CDNSkipCookie = types.StringValue(*cfg.CDNSkipCookie)
	} else {
		state.CDNSkipCookie = types.StringNull()
	}
	if cfg.CacheTagHeader != nil {
		state.CacheTagHeader = types.StringValue(*cfg.CacheTagHeader)
	} else {
		state.CacheTagHeader = types.StringNull()
	}
	if cfg.CacheTagHeaderSeparator != nil {
		state.CacheTagHeaderSeparator = types.StringValue(*cfg.CacheTagHeaderSeparator)
	} else {
		state.CacheTagHeaderSeparator = types.StringNull()
	}

	if cfg.CDNRemoveQueryArgs == nil {
		state.CDNRemoveQueryArgs = types.ListNull(types.StringType)
	} else {
		values := make([]attr.Value, len(cfg.CDNRemoveQueryArgs))
		for i, v := range cfg.CDNRemoveQueryArgs {
			values[i] = types.StringValue(v)
		}
		state.CDNRemoveQueryArgs = types.ListValueMust(types.StringType, values)
	}

	return nil
}
