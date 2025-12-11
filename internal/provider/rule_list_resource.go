package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RuleListResource{}
var _ resource.ResourceWithConfigure = &RuleListResource{}
var _ resource.ResourceWithImportState = &RuleListResource{}

func NewRuleListResource() resource.Resource {
	return &RuleListResource{}
}

type RuleListResource struct {
	client *client.Client
}

type RuleListResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	IPs    types.List   `tfsdk:"ips"`
	Strs   types.List   `tfsdk:"strs"`
	Ints   types.List   `tfsdk:"ints"`
}

func (r *RuleListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_list"
}

func (r *RuleListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a rule list. Lists store collections of IPs, strings, or integers that can be referenced in rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "List identifier (domain/uuid).",
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
			"uuid": schema.StringAttribute{
				Description: "List UUID (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "List name.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "List type: 'ip', 'string', or 'integer'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ips": schema.ListAttribute{
				Description: "IP addresses/networks (for type='ip').",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"strs": schema.ListAttribute{
				Description: "String values (for type='string').",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"ints": schema.ListAttribute{
				Description: "Integer values (for type='integer').",
				ElementType: types.Int64Type,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.Int64Type, []attr.Value{})),
			},
		},
	}
}

func (r *RuleListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RuleListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RuleListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build list
	list := r.buildListFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create list via API
	result, err := r.client.CreateRuleList(plan.Domain.ValueString(), list)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule list",
			"Could not create list for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set computed values
	plan.UUID = types.StringValue(result.UUID)
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + result.UUID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RuleListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get list from API
	list, err := r.client.GetRuleList(state.Domain.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading rule list",
			"Could not read list "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state
	state.Name = types.StringValue(list.Name)
	state.Type = types.StringValue(list.Type)

	// Convert IPs
	if len(list.IPs) > 0 {
		ips := make([]attr.Value, len(list.IPs))
		for i, ip := range list.IPs {
			ips[i] = types.StringValue(ip)
		}
		state.IPs = types.ListValueMust(types.StringType, ips)
	} else {
		state.IPs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Convert Strs
	if len(list.Strs) > 0 {
		strs := make([]attr.Value, len(list.Strs))
		for i, s := range list.Strs {
			strs[i] = types.StringValue(s)
		}
		state.Strs = types.ListValueMust(types.StringType, strs)
	} else {
		state.Strs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Convert Ints
	if len(list.Ints) > 0 {
		ints := make([]attr.Value, len(list.Ints))
		for i, n := range list.Ints {
			ints[i] = types.Int64Value(int64(n))
		}
		state.Ints = types.ListValueMust(types.Int64Type, ints)
	} else {
		state.Ints = types.ListValueMust(types.Int64Type, []attr.Value{})
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RuleListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build list
	list := r.buildListFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update list via API
	err := r.client.UpdateRuleList(plan.Domain.ValueString(), plan.UUID.ValueString(), list)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule list",
			"Could not update list "+plan.UUID.ValueString()+" for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RuleListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete list via API
	err := r.client.DeleteRuleList(state.Domain.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting rule list",
			"Could not delete list "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *RuleListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/uuid
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RuleListResource) buildListFromModel(ctx context.Context, model *RuleListResourceModel, diags *diag.Diagnostics) client.RuleListAdd {
	list := client.RuleListAdd{
		Name: model.Name.ValueString(),
		Type: model.Type.ValueString(),
	}

	// Convert IPs
	if !model.IPs.IsNull() {
		var ips []string
		d := model.IPs.ElementsAs(ctx, &ips, false)
		diags.Append(d...)
		if !diags.HasError() {
			list.IPs = ips
		}
	}

	// Convert Strs
	if !model.Strs.IsNull() {
		var strs []string
		d := model.Strs.ElementsAs(ctx, &strs, false)
		diags.Append(d...)
		if !diags.HasError() {
			list.Strs = strs
		}
	}

	// Convert Ints
	if !model.Ints.IsNull() {
		var ints []int64
		d := model.Ints.ElementsAs(ctx, &ints, false)
		diags.Append(d...)
		if !diags.HasError() {
			list.Ints = make([]int, len(ints))
			for i, n := range ints {
				list.Ints[i] = int(n)
			}
		}
	}

	return list
}
