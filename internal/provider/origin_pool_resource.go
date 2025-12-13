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

var _ resource.Resource = &OriginPoolResource{}
var _ resource.ResourceWithConfigure = &OriginPoolResource{}
var _ resource.ResourceWithImportState = &OriginPoolResource{}

func NewOriginPoolResource() resource.Resource {
	return &OriginPoolResource{}
}

type OriginPoolResource struct {
	client *client.Client
}

type OriginPoolResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Domain                       types.String `tfsdk:"domain"`
	Tag                          types.String `tfsdk:"tag"`
	Addresses                    types.List   `tfsdk:"address"`
	ShieldName                   types.String `tfsdk:"shield_name"`
	LoadBalancingMode            types.String `tfsdk:"load_balancing_mode"`
	LoadBalancingKey             types.String `tfsdk:"load_balancing_key"`
	LoadBalancingOverloadPercent types.Int64  `tfsdk:"load_balancing_overload_percent"`
}

type OriginAddressModel struct {
	Address types.String `tfsdk:"address"`
	Weight  types.Int64  `tfsdk:"weight"`
}

var originAddressTypes = map[string]attr.Type{
	"address": types.StringType,
	"weight":  types.Int64Type,
}

func (r *OriginPoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_origin_pool"
}

func (r *OriginPoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an origin pool for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Origin pool identifier (domain/origins/tag).",
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
			"tag": schema.StringAttribute{
				Description: "Origin pool tag/name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"address": schema.ListNestedAttribute{
				Description: "Backend server addresses.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Description: "Backend address (IP:port, domain:port, or URL).",
							Required:    true,
						},
						"weight": schema.Int64Attribute{
							Description: "Load balancing weight.",
							Optional:    true,
						},
					},
				},
			},
			"shield_name": schema.StringAttribute{
				Description: "Shield location name.",
				Optional:    true,
			},
			"load_balancing_mode": schema.StringAttribute{
				Description: "Load balancing mode (e.g., round_robin, least_conn).",
				Optional:    true,
			},
			"load_balancing_key": schema.StringAttribute{
				Description: "Load balancing key for consistent hashing.",
				Optional:    true,
			},
			"load_balancing_overload_percent": schema.Int64Attribute{
				Description: "Overload percentage for load balancing.",
				Optional:    true,
			},
		},
	}
}

func (r *OriginPoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OriginPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OriginPoolResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build origin pool
	pool := r.buildPoolFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create pool via API
	err := r.client.CreateOriginPool(plan.Domain.ValueString(), pool)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating origin pool",
			"Could not create pool for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set ID
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/origins/" + plan.Tag.ValueString())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *OriginPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OriginPoolResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get pool from API
	pool, err := r.client.GetOriginPool(state.Domain.ValueString(), state.Tag.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading origin pool",
			"Could not read pool "+state.Tag.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state
	state.Tag = types.StringValue(pool.Tag)
	state.ID = types.StringValue(state.Domain.ValueString() + "/origins/" + pool.Tag)

	// Convert addresses
	addresses := make([]attr.Value, len(pool.Addresses))
	for i, addr := range pool.Addresses {
		addrMap := map[string]attr.Value{
			"address": types.StringValue(addr.Address),
		}
		if addr.Weight != nil {
			addrMap["weight"] = types.Int64Value(int64(*addr.Weight))
		} else {
			addrMap["weight"] = types.Int64Null()
		}
		addresses[i] = types.ObjectValueMust(originAddressTypes, addrMap)
	}
	state.Addresses = types.ListValueMust(types.ObjectType{AttrTypes: originAddressTypes}, addresses)

	if pool.ShieldName != nil {
		state.ShieldName = types.StringValue(*pool.ShieldName)
	} else {
		state.ShieldName = types.StringNull()
	}

	if pool.LoadBalancingMode != nil {
		state.LoadBalancingMode = types.StringValue(*pool.LoadBalancingMode)
	} else {
		state.LoadBalancingMode = types.StringNull()
	}

	if pool.LoadBalancingKey != nil {
		state.LoadBalancingKey = types.StringValue(*pool.LoadBalancingKey)
	} else {
		state.LoadBalancingKey = types.StringNull()
	}

	if pool.LoadBalancingOverloadPercent != nil {
		state.LoadBalancingOverloadPercent = types.Int64Value(int64(*pool.LoadBalancingOverloadPercent))
	} else {
		state.LoadBalancingOverloadPercent = types.Int64Null()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OriginPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OriginPoolResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build origin pool
	pool := r.buildPoolFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update pool via API
	err := r.client.UpdateOriginPool(plan.Domain.ValueString(), pool)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating origin pool",
			"Could not update pool for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *OriginPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OriginPoolResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete pool via API
	err := r.client.DeleteOriginPool(state.Domain.ValueString(), state.Tag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting origin pool",
			"Could not delete pool "+state.Tag.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *OriginPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/origins/tag
	parts, err := parseCompositeID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/origins/tag', got %q: %s", req.ID, err),
		)
		return
	}
	if parts[1] != "origins" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/origins/tag', got %q: middle segment must be 'origins'", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *OriginPoolResource) buildPoolFromModel(ctx context.Context, model *OriginPoolResourceModel, diags *diag.Diagnostics) client.OriginPool {
	pool := client.OriginPool{
		Tag: model.Tag.ValueString(),
	}

	// Convert addresses
	var addresses []OriginAddressModel
	d := model.Addresses.ElementsAs(ctx, &addresses, false)
	diags.Append(d...)
	if !diags.HasError() {
		pool.Addresses = make([]client.OriginAddr, len(addresses))
		for i, addr := range addresses {
			pool.Addresses[i] = client.OriginAddr{
				Address: addr.Address.ValueString(),
			}
			if !addr.Weight.IsNull() {
				w := int(addr.Weight.ValueInt64())
				pool.Addresses[i].Weight = &w
			}
		}
	}

	if !model.ShieldName.IsNull() {
		v := model.ShieldName.ValueString()
		pool.ShieldName = &v
	}

	if !model.LoadBalancingMode.IsNull() {
		v := model.LoadBalancingMode.ValueString()
		pool.LoadBalancingMode = &v
	}

	if !model.LoadBalancingKey.IsNull() {
		v := model.LoadBalancingKey.ValueString()
		pool.LoadBalancingKey = &v
	}

	if !model.LoadBalancingOverloadPercent.IsNull() {
		v := int(model.LoadBalancingOverloadPercent.ValueInt64())
		pool.LoadBalancingOverloadPercent = &v
	}

	return pool
}
