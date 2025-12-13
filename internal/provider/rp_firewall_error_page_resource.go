package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPFirewallErrorPageResource{}
var _ resource.ResourceWithConfigure = &RPFirewallErrorPageResource{}
var _ resource.ResourceWithImportState = &RPFirewallErrorPageResource{}

func NewRPFirewallErrorPageResource() resource.Resource {
	return &RPFirewallErrorPageResource{}
}

type RPFirewallErrorPageResource struct {
	client *client.Client
}

type RPFirewallErrorPageResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Content types.String `tfsdk:"content"`
}

func (r *RPFirewallErrorPageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_firewall_error_page"
}

func (r *RPFirewallErrorPageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the RP firewall custom error page. Note: the API does not return the configured content, only whether an error page exists.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP firewall error page identifier (domain/rp_firewall_error_page).",
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
			"enabled": schema.BoolAttribute{
				Description: "Whether a custom error page is configured (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"content": schema.StringAttribute{
				Description: "Custom error page HTML/content to upload. Set to null to clear/disable. Stored in Terraform state (sensitive).",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPFirewallErrorPageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPFirewallErrorPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPFirewallErrorPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyErrorPage(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP firewall error page", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_firewall_error_page")

	if err := r.readErrorPage(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP firewall error page after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPFirewallErrorPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPFirewallErrorPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readErrorPage(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP firewall error page", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_firewall_error_page")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPFirewallErrorPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPFirewallErrorPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyErrorPage(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP firewall error page", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_firewall_error_page")

	if err := r.readErrorPage(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP firewall error page after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPFirewallErrorPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPFirewallErrorPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateRPFirewallErrorPage(state.Domain.ValueString(), map[string]any{"error_page": nil}); err != nil {
		resp.Diagnostics.AddError("Error deleting RP firewall error page", err.Error())
		return
	}
}

func (r *RPFirewallErrorPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPFirewallErrorPageResource) applyErrorPage(ctx context.Context, model *RPFirewallErrorPageResourceModel, diags *diag.Diagnostics) error {
	if model.Content.IsUnknown() {
		return nil
	}

	var body map[string]any
	if model.Content.IsNull() {
		body = map[string]any{"error_page": nil}
	} else {
		body = map[string]any{"error_page": model.Content.ValueString()}
	}

	return r.client.UpdateRPFirewallErrorPage(model.Domain.ValueString(), body)
}

func (r *RPFirewallErrorPageResource) readErrorPage(ctx context.Context, state *RPFirewallErrorPageResourceModel) error {
	cfg, err := r.client.GetRPFirewallErrorPage(state.Domain.ValueString())
	if err != nil {
		return err
	}

	state.Enabled = types.BoolValue(cfg.ErrorPage)
	// state.Content is intentionally preserved from prior state (API does not return content).
	return nil
}
