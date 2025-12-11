package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RateLimitZoneResource{}
var _ resource.ResourceWithConfigure = &RateLimitZoneResource{}
var _ resource.ResourceWithImportState = &RateLimitZoneResource{}

func NewRateLimitZoneResource() resource.Resource {
	return &RateLimitZoneResource{}
}

type RateLimitZoneResource struct {
	client *client.Client
}

type RateLimitZoneResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Domain                    types.String `tfsdk:"domain"`
	Name                      types.String `tfsdk:"name"`
	BlockDurationSec          types.Int64  `tfsdk:"block_duration_sec"`
	ConcurrentConnections     types.Int64  `tfsdk:"concurrent_connections"`
	ConnectionsMax            types.Int64  `tfsdk:"connections_max"`
	ConnectionsIntervalSec    types.Int64  `tfsdk:"connections_interval_sec"`
	RequestsMax               types.Int64  `tfsdk:"requests_max"`
	RequestsIntervalSec       types.Int64  `tfsdk:"requests_interval_sec"`
	ResponseErrorsMax         types.Int64  `tfsdk:"response_errors_max"`
	ResponseErrorsIntervalSec types.Int64  `tfsdk:"response_errors_interval_sec"`
}

func (r *RateLimitZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rate_limit_zone"
}

func (r *RateLimitZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a rate limit zone. Zones define rate limiting rules that can be referenced in rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Zone identifier (domain/name).",
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
			"name": schema.StringAttribute{
				Description: "Zone name (used to reference in rules).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"block_duration_sec": schema.Int64Attribute{
				Description: "How long to block when limit exceeded (seconds).",
				Optional:    true,
			},
			"concurrent_connections": schema.Int64Attribute{
				Description: "Maximum concurrent connections.",
				Optional:    true,
			},
			"connections_max": schema.Int64Attribute{
				Description: "Maximum connections in interval.",
				Optional:    true,
			},
			"connections_interval_sec": schema.Int64Attribute{
				Description: "Time window for connection limit (seconds).",
				Optional:    true,
			},
			"requests_max": schema.Int64Attribute{
				Description: "Maximum requests in interval.",
				Optional:    true,
			},
			"requests_interval_sec": schema.Int64Attribute{
				Description: "Time window for request limit (seconds).",
				Optional:    true,
			},
			"response_errors_max": schema.Int64Attribute{
				Description: "Maximum response errors in interval.",
				Optional:    true,
			},
			"response_errors_interval_sec": schema.Int64Attribute{
				Description: "Time window for response error limit (seconds).",
				Optional:    true,
			},
		},
	}
}

func (r *RateLimitZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RateLimitZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RateLimitZoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build zone
	zone := r.buildZoneFromModel(&plan)

	// Create zone via API
	created, err := r.client.CreateRateLimitZone(plan.Domain.ValueString(), zone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rate limit zone",
			"Could not create zone for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state from response
	r.updateModelFromZone(&plan, created)
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + plan.Name.ValueString())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RateLimitZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RateLimitZoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get zone from API
	zone, err := r.client.GetRateLimitZone(state.Domain.ValueString(), state.Name.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading rate limit zone",
			"Could not read zone "+state.Name.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state
	r.updateModelFromZone(&state, zone)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RateLimitZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RateLimitZoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build zone
	zone := r.buildZoneFromModel(&plan)

	// Update zone via API
	err := r.client.UpdateRateLimitZone(plan.Domain.ValueString(), plan.Name.ValueString(), zone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rate limit zone",
			"Could not update zone "+plan.Name.ValueString()+" for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RateLimitZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RateLimitZoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete zone via API
	err := r.client.DeleteRateLimitZone(state.Domain.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting rate limit zone",
			"Could not delete zone "+state.Name.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *RateLimitZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/name
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RateLimitZoneResource) buildZoneFromModel(model *RateLimitZoneResourceModel) client.RateLimitZone {
	zone := client.RateLimitZone{
		Name: model.Name.ValueString(),
	}

	if !model.BlockDurationSec.IsNull() {
		v := int(model.BlockDurationSec.ValueInt64())
		zone.BlockDurationSec = &v
	}
	if !model.ConcurrentConnections.IsNull() {
		v := int(model.ConcurrentConnections.ValueInt64())
		zone.ConcurrentConnections = &v
	}
	if !model.ConnectionsMax.IsNull() {
		v := int(model.ConnectionsMax.ValueInt64())
		zone.ConnectionsMax = &v
	}
	if !model.ConnectionsIntervalSec.IsNull() {
		v := int(model.ConnectionsIntervalSec.ValueInt64())
		zone.ConnectionsIntervalSec = &v
	}
	if !model.RequestsMax.IsNull() {
		v := int(model.RequestsMax.ValueInt64())
		zone.RequestsMax = &v
	}
	if !model.RequestsIntervalSec.IsNull() {
		v := int(model.RequestsIntervalSec.ValueInt64())
		zone.RequestsIntervalSec = &v
	}
	if !model.ResponseErrorsMax.IsNull() {
		v := int(model.ResponseErrorsMax.ValueInt64())
		zone.ResponseErrorsMax = &v
	}
	if !model.ResponseErrorsIntervalSec.IsNull() {
		v := int(model.ResponseErrorsIntervalSec.ValueInt64())
		zone.ResponseErrorsIntervalSec = &v
	}

	return zone
}

func (r *RateLimitZoneResource) updateModelFromZone(model *RateLimitZoneResourceModel, zone *client.RateLimitZone) {
	model.Name = types.StringValue(zone.Name)

	if zone.BlockDurationSec != nil {
		model.BlockDurationSec = types.Int64Value(int64(*zone.BlockDurationSec))
	} else {
		model.BlockDurationSec = types.Int64Null()
	}

	if zone.ConcurrentConnections != nil {
		model.ConcurrentConnections = types.Int64Value(int64(*zone.ConcurrentConnections))
	} else {
		model.ConcurrentConnections = types.Int64Null()
	}

	if zone.ConnectionsMax != nil {
		model.ConnectionsMax = types.Int64Value(int64(*zone.ConnectionsMax))
	} else {
		model.ConnectionsMax = types.Int64Null()
	}

	if zone.ConnectionsIntervalSec != nil {
		model.ConnectionsIntervalSec = types.Int64Value(int64(*zone.ConnectionsIntervalSec))
	} else {
		model.ConnectionsIntervalSec = types.Int64Null()
	}

	if zone.RequestsMax != nil {
		model.RequestsMax = types.Int64Value(int64(*zone.RequestsMax))
	} else {
		model.RequestsMax = types.Int64Null()
	}

	if zone.RequestsIntervalSec != nil {
		model.RequestsIntervalSec = types.Int64Value(int64(*zone.RequestsIntervalSec))
	} else {
		model.RequestsIntervalSec = types.Int64Null()
	}

	if zone.ResponseErrorsMax != nil {
		model.ResponseErrorsMax = types.Int64Value(int64(*zone.ResponseErrorsMax))
	} else {
		model.ResponseErrorsMax = types.Int64Null()
	}

	if zone.ResponseErrorsIntervalSec != nil {
		model.ResponseErrorsIntervalSec = types.Int64Value(int64(*zone.ResponseErrorsIntervalSec))
	} else {
		model.ResponseErrorsIntervalSec = types.Int64Null()
	}
}
