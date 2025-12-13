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

var _ resource.Resource = &RPWAFRuleGroupResource{}
var _ resource.ResourceWithConfigure = &RPWAFRuleGroupResource{}
var _ resource.ResourceWithImportState = &RPWAFRuleGroupResource{}

func NewRPWAFRuleGroupResource() resource.Resource {
	return &RPWAFRuleGroupResource{}
}

type RPWAFRuleGroupResource struct {
	client *client.Client
}

type RPWAFRuleGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	Ruleset     types.String `tfsdk:"ruleset"`
	FileName    types.String `tfsdk:"file_name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func (r *RPWAFRuleGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_waf_rule_group"
}

func (r *RPWAFRuleGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the enabled state of a WAF rule group (file) within a ruleset for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Rule group identifier (domain/ruleset/<ruleset>/rulegroup/<file_name>).",
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
			"ruleset": schema.StringAttribute{
				Description: "Ruleset name (e.g. owaspv33).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_name": schema.StringAttribute{
				Description: "Rule group file name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Rule group description (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule group is enabled.",
				Required:    true,
			},
		},
	}
}

func (r *RPWAFRuleGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPWAFRuleGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPWAFRuleGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetRPWAFRuleGroupEnabled(plan.Domain.ValueString(), plan.Ruleset.ValueString(), plan.FileName.ValueString(), plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error setting WAF rule group state", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/ruleset/%s/rulegroup/%s", plan.Domain.ValueString(), plan.Ruleset.ValueString(), plan.FileName.ValueString()))
	if err := r.read(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading WAF rule group after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFRuleGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPWAFRuleGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.read(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WAF rule group", err.Error())
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/ruleset/%s/rulegroup/%s", state.Domain.ValueString(), state.Ruleset.ValueString(), state.FileName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPWAFRuleGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPWAFRuleGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetRPWAFRuleGroupEnabled(plan.Domain.ValueString(), plan.Ruleset.ValueString(), plan.FileName.ValueString(), plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error setting WAF rule group state", err.Error())
		return
	}

	if err := r.read(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading WAF rule group after update", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/ruleset/%s/rulegroup/%s", plan.Domain.ValueString(), plan.Ruleset.ValueString(), plan.FileName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFRuleGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPWAFRuleGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset by re-enabling the group.
	if err := r.client.SetRPWAFRuleGroupEnabled(state.Domain.ValueString(), state.Ruleset.ValueString(), state.FileName.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting WAF rule group state", err.Error())
		return
	}
}

func (r *RPWAFRuleGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, 5)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/ruleset/<ruleset>/rulegroup/<file_name>', got %q: %s", req.ID, err),
		)
		return
	}
	if parts[1] != "ruleset" || parts[3] != "rulegroup" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/ruleset/<ruleset>/rulegroup/<file_name>', got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ruleset"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_name"), parts[4])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *RPWAFRuleGroupResource) read(ctx context.Context, model *RPWAFRuleGroupResourceModel) error {
	groups, err := r.client.ListRPWAFRuleGroups(model.Domain.ValueString(), model.Ruleset.ValueString())
	if err != nil {
		return err
	}

	for _, g := range groups {
		if g.FileName != model.FileName.ValueString() {
			continue
		}
		model.Enabled = types.BoolValue(g.Enabled)
		if g.Description == nil {
			model.Description = types.StringNull()
		} else {
			model.Description = types.StringValue(*g.Description)
		}
		return nil
	}

	return &client.APIError{StatusCode: 404, Body: "rule group not found"}
}
