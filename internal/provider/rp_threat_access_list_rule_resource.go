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

var _ resource.Resource = &RPThreatAccessListRuleResource{}
var _ resource.ResourceWithConfigure = &RPThreatAccessListRuleResource{}
var _ resource.ResourceWithImportState = &RPThreatAccessListRuleResource{}

func NewRPThreatAccessListRuleResource() resource.Resource {
	return &RPThreatAccessListRuleResource{}
}

type RPThreatAccessListRuleResource struct {
	client *client.Client
}

type RPThreatAccessListRuleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	UUID        types.String `tfsdk:"uuid"`
	RuleType    types.String `tfsdk:"rule_type"`
	Content     types.String `tfsdk:"content"`
	Description types.String `tfsdk:"description"`
	ContentType types.String `tfsdk:"content_type"`
	Created     types.String `tfsdk:"created"`
}

func (r *RPThreatAccessListRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_threat_access_list_rule"
}

func (r *RPThreatAccessListRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a reverse proxy threat access list rule for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Access list rule identifier (domain/access_list/uuid).",
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
				Description: "Access list rule UUID (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_type": schema.StringAttribute{
				Description: "Rule type (spec enum: preset, blacklist, whitelist).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Description: "Rule content value (IP, country code, ASN, or preset name depending on type).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Rule description.",
				Required:    true,
			},
			"content_type": schema.StringAttribute{
				Description: "Content type derived by the API (spec enum: preset, ip, cc, asn).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.StringAttribute{
				Description: "Creation timestamp (RFC3339, computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPThreatAccessListRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPThreatAccessListRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPThreatAccessListRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateThreatAccessListRule(plan.Domain.ValueString(), client.AccessListRuleAdd{
		RuleType:    plan.RuleType.ValueString(),
		Content:     plan.Content.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating threat access list rule", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	plan.ContentType = types.StringValue(created.ContentType)
	plan.Created = types.StringValue(created.Created)
	plan.ID = types.StringValue(fmt.Sprintf("%s/access_list/%s", plan.Domain.ValueString(), created.UUID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPThreatAccessListRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPThreatAccessListRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetThreatAccessListRule(state.Domain.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading threat access list rule", err.Error())
		return
	}

	state.RuleType = types.StringValue(rule.RuleType)
	state.Content = types.StringValue(rule.Content)
	state.Description = types.StringValue(rule.Description)
	state.ContentType = types.StringValue(rule.ContentType)
	state.Created = types.StringValue(rule.Created)
	state.ID = types.StringValue(fmt.Sprintf("%s/access_list/%s", state.Domain.ValueString(), state.UUID.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPThreatAccessListRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPThreatAccessListRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateThreatAccessListRule(plan.Domain.ValueString(), plan.UUID.ValueString(), client.AccessListRuleUpdate{
		Description: plan.Description.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error updating threat access list rule", err.Error())
		return
	}

	rule, err := r.client.GetThreatAccessListRule(plan.Domain.ValueString(), plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading threat access list rule after update", err.Error())
		return
	}

	plan.RuleType = types.StringValue(rule.RuleType)
	plan.Content = types.StringValue(rule.Content)
	plan.Description = types.StringValue(rule.Description)
	plan.ContentType = types.StringValue(rule.ContentType)
	plan.Created = types.StringValue(rule.Created)
	plan.ID = types.StringValue(fmt.Sprintf("%s/access_list/%s", plan.Domain.ValueString(), plan.UUID.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPThreatAccessListRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPThreatAccessListRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteThreatAccessListRule(state.Domain.ValueString(), state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting threat access list rule", err.Error())
		return
	}
}

func (r *RPThreatAccessListRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/access_list/uuid', got %q: %s", req.ID, err),
		)
		return
	}
	if parts[1] != "access_list" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/access_list/uuid', got %q (segment 2 must be 'access_list')", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
