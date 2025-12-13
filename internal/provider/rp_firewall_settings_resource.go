package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPFirewallSettingsResource{}
var _ resource.ResourceWithConfigure = &RPFirewallSettingsResource{}
var _ resource.ResourceWithImportState = &RPFirewallSettingsResource{}

func NewRPFirewallSettingsResource() resource.Resource {
	return &RPFirewallSettingsResource{}
}

type RPFirewallSettingsResource struct {
	client *client.Client
}

type RPFirewallSettingsResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	ChallengeCookieKey types.List   `tfsdk:"challenge_cookie_key"`
}

func (r *RPFirewallSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_firewall_settings"
}

func (r *RPFirewallSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages RP firewall settings for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP firewall settings identifier (domain/rp_firewall_settings).",
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
			"challenge_cookie_key": schema.ListAttribute{
				Description: "Challenge cookie key types (spec enum: fingerprint_tls, ip, session_id). Set to null to clear.",
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

func (r *RPFirewallSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPFirewallSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPFirewallSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyFirewallSettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP firewall settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_firewall_settings")

	if err := r.readFirewallSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP firewall settings after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPFirewallSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPFirewallSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readFirewallSettings(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP firewall settings", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_firewall_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPFirewallSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPFirewallSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyFirewallSettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP firewall settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_firewall_settings")

	if err := r.readFirewallSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP firewall settings after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPFirewallSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPFirewallSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"challenge_cookie_key": nil,
	}

	if err := r.client.UpdateRPFirewallSettings(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting RP firewall settings", err.Error())
		return
	}
}

func (r *RPFirewallSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPFirewallSettingsResource) applyFirewallSettings(ctx context.Context, model *RPFirewallSettingsResourceModel, diags *diag.Diagnostics) error {
	update := map[string]any{}

	if !model.ChallengeCookieKey.IsUnknown() {
		if model.ChallengeCookieKey.IsNull() {
			update["challenge_cookie_key"] = nil
		} else {
			var keys []string
			diags.Append(model.ChallengeCookieKey.ElementsAs(ctx, &keys, false)...)
			if diags.HasError() {
				return fmt.Errorf("invalid challenge_cookie_key list")
			}
			update["challenge_cookie_key"] = keys
		}
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPFirewallSettings(model.Domain.ValueString(), update)
}

func (r *RPFirewallSettingsResource) readFirewallSettings(ctx context.Context, state *RPFirewallSettingsResourceModel) error {
	cfg, err := r.client.GetRPFirewallSettings(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if cfg.ChallengeCookieKey == nil {
		state.ChallengeCookieKey = types.ListNull(types.StringType)
	} else {
		values := make([]attr.Value, len(cfg.ChallengeCookieKey))
		for i, v := range cfg.ChallengeCookieKey {
			values[i] = types.StringValue(v)
		}
		state.ChallengeCookieKey = types.ListValueMust(types.StringType, values)
	}

	return nil
}
