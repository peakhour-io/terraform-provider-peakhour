package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RuleResource{}
var _ resource.ResourceWithConfigure = &RuleResource{}
var _ resource.ResourceWithImportState = &RuleResource{}

func NewRuleResource() resource.Resource {
	return &RuleResource{}
}

type RuleResource struct {
	client *client.Client
}

type RuleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	UUID        types.String `tfsdk:"uuid"`
	Phase       types.String `tfsdk:"phase"`
	Name        types.String `tfsdk:"name"`
	FilterStr   types.String `tfsdk:"filter_str"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	ActionsJSON types.String `tfsdk:"actions_json"`
}

func (r *RuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

func (r *RuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a rule in a specific phase for a domain. Rules allow conditional logic and actions based on request properties.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Rule identifier (domain/uuid).",
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
				Description: "Rule UUID (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"phase": schema.StringAttribute{
				Description: "Phase name (e.g., 'firewall', 'request_headers', 'url_config', 'rate_limit_request').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Rule name.",
				Required:    true,
			},
			"filter_str": schema.StringAttribute{
				Description: "Filter expression (Wirefilter syntax, e.g., 'http.request.uri.path matches \"^/api/\"').",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"actions_json": schema.StringAttribute{
				Description: "Actions as JSON string. Structure: map[action_type][]action. Example: '{\"firewall\":[{\"type\":\"firewall\",\"action\":\"deny\"}]}'",
				Required:    true,
			},
		},
	}
}

func (r *RuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RuleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse actions JSON
	var actions map[string][]map[string]any
	if err := json.Unmarshal([]byte(plan.ActionsJSON.ValueString()), &actions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid actions JSON",
			"Could not parse actions_json: "+err.Error(),
		)
		return
	}

	// Create rule
	ruleAdd := client.RulePhaseAdd{
		Phase:     plan.Phase.ValueString(),
		Name:      plan.Name.ValueString(),
		FilterStr: plan.FilterStr.ValueString(),
		Actions:   actions,
	}

	result, err := r.client.CreateRule(plan.Domain.ValueString(), ruleAdd)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set computed values
	plan.UUID = types.StringValue(result.UUID)
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + result.UUID)

	// If enabled not set, default to true
	if plan.Enabled.IsNull() {
		plan.Enabled = types.BoolValue(true)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RuleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get rule from API
	rule, err := r.client.GetRule(state.Domain.ValueString(), state.Phase.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading rule",
			"Could not read rule "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state
	state.Name = types.StringValue(rule.Name)
	state.FilterStr = types.StringValue(rule.FilterStr)
	state.Phase = types.StringValue(rule.Phase)
	state.Enabled = types.BoolValue(rule.Enabled)

	// Convert actions to JSON
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error serializing actions",
			"Could not serialize actions to JSON: "+err.Error(),
		)
		return
	}
	state.ActionsJSON = types.StringValue(string(actionsJSON))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RuleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse actions JSON
	var actions map[string][]map[string]any
	if err := json.Unmarshal([]byte(plan.ActionsJSON.ValueString()), &actions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid actions JSON",
			"Could not parse actions_json: "+err.Error(),
		)
		return
	}

	// Build update
	name := plan.Name.ValueString()
	filterStr := plan.FilterStr.ValueString()
	enabled := plan.Enabled.ValueBool()

	update := client.RulePhaseUpdate{
		Name:      &name,
		FilterStr: &filterStr,
		Enabled:   &enabled,
		Actions:   actions,
	}

	// Update rule via API
	err := r.client.UpdateRule(plan.Domain.ValueString(), plan.Phase.ValueString(), plan.UUID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not update rule "+plan.UUID.ValueString()+" for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RuleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete rule via API
	err := r.client.DeleteRule(state.Domain.ValueString(), state.Phase.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting rule",
			"Could not delete rule "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *RuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/phase/uuid
	parts, err := parseCompositeID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/phase/uuid', got %q: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("phase"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
