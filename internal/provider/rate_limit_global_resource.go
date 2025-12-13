package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RateLimitGlobalResource{}
var _ resource.ResourceWithConfigure = &RateLimitGlobalResource{}
var _ resource.ResourceWithImportState = &RateLimitGlobalResource{}

func NewRateLimitGlobalResource() resource.Resource {
	return &RateLimitGlobalResource{}
}

type RateLimitGlobalResource struct {
	client *client.Client
}

type RateLimitGlobalResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Domain                 types.String `tfsdk:"domain"`
	BlockDurationSec       types.Int64  `tfsdk:"block_duration_sec"`
	ConcurrentConnections  types.Int64  `tfsdk:"concurrent_connections"`
	ConnectionsMax         types.Int64  `tfsdk:"connections_max"`
	ConnectionsIntervalSec types.Int64  `tfsdk:"connections_interval_sec"`
	RequestsMax            types.Int64  `tfsdk:"requests_max"`
	RequestsIntervalSec    types.Int64  `tfsdk:"requests_interval_sec"`
	ResponseErrorsMax      types.Int64  `tfsdk:"response_errors_max"`
	ResponseErrorsInterval types.Int64  `tfsdk:"response_errors_interval_sec"`
}

func (r *RateLimitGlobalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rate_limit_global"
}

func (r *RateLimitGlobalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	useState := []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		Description: "Manages global rate limit settings for a domain. Global settings apply account-wide limits that are not tied to rule zones.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Global rate limit identifier (domain/rate_limit_global).",
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
			"block_duration_sec": schema.Int64Attribute{
				Description:   "How long to block when limit exceeded (seconds).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"concurrent_connections": schema.Int64Attribute{
				Description:   "Maximum concurrent connections.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"connections_max": schema.Int64Attribute{
				Description:   "Maximum connections in interval.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"connections_interval_sec": schema.Int64Attribute{
				Description:   "Time window for connection limit (seconds).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"requests_max": schema.Int64Attribute{
				Description:   "Maximum requests in interval.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"requests_interval_sec": schema.Int64Attribute{
				Description:   "Time window for request limit (seconds).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"response_errors_max": schema.Int64Attribute{
				Description:   "Maximum response errors in interval.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
			"response_errors_interval_sec": schema.Int64Attribute{
				Description:   "Time window for response error limit (seconds).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useState,
			},
		},
	}
}

func (r *RateLimitGlobalResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RateLimitGlobalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RateLimitGlobalResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	global := buildRateLimitGlobalFromModel(&plan)

	err := r.client.UpdateRateLimitGlobal(plan.Domain.ValueString(), global)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating global rate limit settings",
			"Could not update global rate limit for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rate_limit_global")
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RateLimitGlobalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RateLimitGlobalResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rl, err := r.client.GetRateLimit(state.Domain.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading global rate limit settings",
			"Could not read global rate limit for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	updateRateLimitGlobalModelFromAPI(&state, &rl.Global)
	state.ID = types.StringValue(state.Domain.ValueString() + "/rate_limit_global")

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RateLimitGlobalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RateLimitGlobalResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	global := buildRateLimitGlobalFromModel(&plan)

	err := r.client.UpdateRateLimitGlobal(plan.Domain.ValueString(), global)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating global rate limit settings",
			"Could not update global rate limit for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RateLimitGlobalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RateLimitGlobalResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset to defaults by sending empty global settings.
	err := r.client.UpdateRateLimitGlobal(state.Domain.ValueString(), client.RateLimitGlobal{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting global rate limit settings",
			"Could not reset global rate limit for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *RateLimitGlobalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func buildRateLimitGlobalFromModel(model *RateLimitGlobalResourceModel) client.RateLimitGlobal {
	global := client.RateLimitGlobal{}

	if v, ok := optionalInt64(model.BlockDurationSec); ok {
		global.BlockDurationSec = &v
	}
	if v, ok := optionalInt64(model.ConcurrentConnections); ok {
		global.ConcurrentConnections = &v
	}
	if v, ok := optionalInt64(model.ConnectionsMax); ok {
		global.ConnectionsMax = &v
	}
	if v, ok := optionalInt64(model.ConnectionsIntervalSec); ok {
		global.ConnectionsIntervalSec = &v
	}
	if v, ok := optionalInt64(model.RequestsMax); ok {
		global.RequestsMax = &v
	}
	if v, ok := optionalInt64(model.RequestsIntervalSec); ok {
		global.RequestsIntervalSec = &v
	}
	if v, ok := optionalInt64(model.ResponseErrorsMax); ok {
		global.ResponseErrorsMax = &v
	}
	if v, ok := optionalInt64(model.ResponseErrorsInterval); ok {
		global.ResponseErrorsIntervalSec = &v
	}

	return global
}

func updateRateLimitGlobalModelFromAPI(model *RateLimitGlobalResourceModel, global *client.RateLimitGlobal) {
	if global.BlockDurationSec != nil {
		model.BlockDurationSec = types.Int64Value(int64(*global.BlockDurationSec))
	} else {
		model.BlockDurationSec = types.Int64Null()
	}
	if global.ConcurrentConnections != nil {
		model.ConcurrentConnections = types.Int64Value(int64(*global.ConcurrentConnections))
	} else {
		model.ConcurrentConnections = types.Int64Null()
	}
	if global.ConnectionsMax != nil {
		model.ConnectionsMax = types.Int64Value(int64(*global.ConnectionsMax))
	} else {
		model.ConnectionsMax = types.Int64Null()
	}
	if global.ConnectionsIntervalSec != nil {
		model.ConnectionsIntervalSec = types.Int64Value(int64(*global.ConnectionsIntervalSec))
	} else {
		model.ConnectionsIntervalSec = types.Int64Null()
	}
	if global.RequestsMax != nil {
		model.RequestsMax = types.Int64Value(int64(*global.RequestsMax))
	} else {
		model.RequestsMax = types.Int64Null()
	}
	if global.RequestsIntervalSec != nil {
		model.RequestsIntervalSec = types.Int64Value(int64(*global.RequestsIntervalSec))
	} else {
		model.RequestsIntervalSec = types.Int64Null()
	}
	if global.ResponseErrorsMax != nil {
		model.ResponseErrorsMax = types.Int64Value(int64(*global.ResponseErrorsMax))
	} else {
		model.ResponseErrorsMax = types.Int64Null()
	}
	if global.ResponseErrorsIntervalSec != nil {
		model.ResponseErrorsInterval = types.Int64Value(int64(*global.ResponseErrorsIntervalSec))
	} else {
		model.ResponseErrorsInterval = types.Int64Null()
	}
}

func optionalInt64(v types.Int64) (int, bool) {
	if v.IsNull() || v.IsUnknown() {
		return 0, false
	}
	return int(v.ValueInt64()), true
}
