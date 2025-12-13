package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPWAFCustomRuleResource{}
var _ resource.ResourceWithConfigure = &RPWAFCustomRuleResource{}
var _ resource.ResourceWithImportState = &RPWAFCustomRuleResource{}

func NewRPWAFCustomRuleResource() resource.Resource {
	return &RPWAFCustomRuleResource{}
}

type RPWAFCustomRuleResource struct {
	client *client.Client
}

type RPWAFCustomRuleResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	UUID    types.String `tfsdk:"uuid"`
	RuleID  types.Int64  `tfsdk:"rule_id"`
	Created types.String `tfsdk:"created"`

	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`

	RulesJSON   types.String `tfsdk:"rules_json"`
	ActionJSON  types.String `tfsdk:"action_json"`
	LoggingJSON types.String `tfsdk:"logging_json"`
}

func (r *RPWAFCustomRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_waf_custom_rule"
}

func (r *RPWAFCustomRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a reverse proxy WAF custom rule for a domain. Complex fields are represented as JSON.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Custom rule identifier (domain/customrule/uuid).",
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
				Description: "Custom rule UUID (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_id": schema.Int64Attribute{
				Description: "Numeric rule ID (computed).",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "Creation timestamp (RFC3339, computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Custom rule name. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Custom rule description. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"rules_json": schema.StringAttribute{
				Description: "Rules as JSON array (WafCustomRuleExpression[]).",
				Required:    true,
			},
			"action_json": schema.StringAttribute{
				Description: "Action as JSON object (WafAction).",
				Required:    true,
			},
			"logging_json": schema.StringAttribute{
				Description: "Logging as JSON object (WafLogging).",
				Required:    true,
			},
		},
	}
}

func (r *RPWAFCustomRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPWAFCustomRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPWAFCustomRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildRuleBody(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateRPWAFCustomRule(plan.Domain.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating WAF custom rule", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	plan.RuleID = types.Int64Value(int64(created.RuleID))
	plan.Created = types.StringValue(created.Created)
	plan.ID = types.StringValue(fmt.Sprintf("%s/customrule/%s", plan.Domain.ValueString(), created.UUID))

	if err := r.read(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF custom rule after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFCustomRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPWAFCustomRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.read(ctx, &state, &resp.Diagnostics); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WAF custom rule", err.Error())
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/customrule/%s", state.Domain.ValueString(), state.UUID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPWAFCustomRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPWAFCustomRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildRuleBody(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateRPWAFCustomRule(plan.Domain.ValueString(), plan.UUID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Error updating WAF custom rule", err.Error())
		return
	}

	if err := r.read(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF custom rule after update", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/customrule/%s", plan.Domain.ValueString(), plan.UUID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFCustomRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPWAFCustomRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRPWAFCustomRule(state.Domain.ValueString(), state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting WAF custom rule", err.Error())
		return
	}
}

func (r *RPWAFCustomRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/customrule/uuid', got %q: %s", req.ID, err),
		)
		return
	}
	if parts[1] != "customrule" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/customrule/uuid', got %q (segment 2 must be 'customrule')", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *RPWAFCustomRuleResource) read(ctx context.Context, model *RPWAFCustomRuleResourceModel, diags *diag.Diagnostics) error {
	rules, err := r.client.ListRPWAFCustomRules(model.Domain.ValueString())
	if err != nil {
		return err
	}

	var found *client.WAFCustomRule
	for i := range rules {
		if rules[i].UUID == model.UUID.ValueString() {
			found = &rules[i]
			break
		}
	}
	if found == nil {
		return &client.APIError{StatusCode: 404, Body: "custom rule not found"}
	}

	if found.Name == nil {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(*found.Name)
	}
	if found.Description == nil {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(*found.Description)
	}
	if found.Enabled == nil {
		model.Enabled = types.BoolNull()
	} else {
		model.Enabled = types.BoolValue(*found.Enabled)
	}

	model.RuleID = types.Int64Value(int64(found.RuleID))
	model.Created = types.StringValue(found.Created)

	rulesRaw, err := json.Marshal(found.Rules)
	if err != nil {
		diags.AddError("Error marshalling rules", err.Error())
		return nil
	}
	rulesJSON, err := normalizeJSON(string(rulesRaw))
	if err != nil {
		diags.AddError("Error normalizing rules_json", err.Error())
		return nil
	}
	model.RulesJSON = types.StringValue(rulesJSON)

	actionRaw, err := json.Marshal(found.Action)
	if err != nil {
		diags.AddError("Error marshalling action", err.Error())
		return nil
	}
	actionJSON, err := normalizeJSON(string(actionRaw))
	if err != nil {
		diags.AddError("Error normalizing action_json", err.Error())
		return nil
	}
	model.ActionJSON = types.StringValue(actionJSON)

	loggingRaw, err := json.Marshal(found.Logging)
	if err != nil {
		diags.AddError("Error marshalling logging", err.Error())
		return nil
	}
	loggingJSON, err := normalizeJSON(string(loggingRaw))
	if err != nil {
		diags.AddError("Error normalizing logging_json", err.Error())
		return nil
	}
	model.LoggingJSON = types.StringValue(loggingJSON)

	return nil
}

func (r *RPWAFCustomRuleResource) buildRuleBody(model *RPWAFCustomRuleResourceModel, diags *diag.Diagnostics) map[string]any {
	body := map[string]any{}

	if !model.Name.IsUnknown() {
		if model.Name.IsNull() {
			body["name"] = nil
		} else {
			body["name"] = model.Name.ValueString()
		}
	}

	if !model.Description.IsUnknown() {
		if model.Description.IsNull() {
			body["description"] = nil
		} else {
			body["description"] = model.Description.ValueString()
		}
	}

	if !model.Enabled.IsUnknown() {
		if model.Enabled.IsNull() {
			body["enabled"] = nil
		} else {
			body["enabled"] = model.Enabled.ValueBool()
		}
	}

	rulesJSON, err := normalizeJSON(model.RulesJSON.ValueString())
	if err != nil {
		diags.AddError("Invalid rules_json", err.Error())
		return nil
	}
	var rules []any
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		diags.AddError("Invalid rules_json", err.Error())
		return nil
	}
	body["rules"] = rules
	model.RulesJSON = types.StringValue(rulesJSON)

	actionJSON, err := normalizeJSON(model.ActionJSON.ValueString())
	if err != nil {
		diags.AddError("Invalid action_json", err.Error())
		return nil
	}
	var action map[string]any
	if err := json.Unmarshal([]byte(actionJSON), &action); err != nil {
		diags.AddError("Invalid action_json", err.Error())
		return nil
	}
	body["action"] = action
	model.ActionJSON = types.StringValue(actionJSON)

	loggingJSON, err := normalizeJSON(model.LoggingJSON.ValueString())
	if err != nil {
		diags.AddError("Invalid logging_json", err.Error())
		return nil
	}
	var logging map[string]any
	if err := json.Unmarshal([]byte(loggingJSON), &logging); err != nil {
		diags.AddError("Invalid logging_json", err.Error())
		return nil
	}
	body["logging"] = logging
	model.LoggingJSON = types.StringValue(loggingJSON)

	return body
}
