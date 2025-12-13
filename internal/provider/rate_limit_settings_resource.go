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

var _ resource.Resource = &RateLimitSettingsResource{}
var _ resource.ResourceWithConfigure = &RateLimitSettingsResource{}
var _ resource.ResourceWithImportState = &RateLimitSettingsResource{}

func NewRateLimitSettingsResource() resource.Resource {
	return &RateLimitSettingsResource{}
}

type RateLimitSettingsResource struct {
	client *client.Client
}

type RateLimitSettingsResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	Mode   types.List   `tfsdk:"mode"`
}

func (r *RateLimitSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rate_limit_settings"
}

func (r *RateLimitSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages rate limit mode settings for a domain (e.g., global, vhost, zone). This controls which rate limiting systems are active.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Rate limit settings identifier (domain/rate_limit_settings).",
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
			"mode": schema.ListAttribute{
				Description: "Enabled rate limit modes (spec enum: none, global, vhost, vhost-busy, zone, client-errors, server-errors, all).",
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

func (r *RateLimitSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RateLimitSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RateLimitSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyRateLimitSettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating rate limit settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rate_limit_settings")

	if err := r.readRateLimitSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading rate limit settings after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RateLimitSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RateLimitSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readRateLimitSettings(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading rate limit settings", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rate_limit_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RateLimitSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RateLimitSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyRateLimitSettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating rate limit settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rate_limit_settings")

	if err := r.readRateLimitSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading rate limit settings after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RateLimitSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RateLimitSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset by disabling modes.
	if err := r.client.UpdateRateLimitSettings(state.Domain.ValueString(), client.RateLimitSettings{Mode: []string{"none"}}); err != nil {
		resp.Diagnostics.AddError("Error deleting rate limit settings", err.Error())
		return
	}
}

func (r *RateLimitSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RateLimitSettingsResource) applyRateLimitSettings(ctx context.Context, model *RateLimitSettingsResourceModel, diags *diag.Diagnostics) error {
	// If mode is unknown, do not change remote settings.
	if model.Mode.IsUnknown() {
		return nil
	}

	var modes []string
	if model.Mode.IsNull() {
		modes = []string{"none"}
	} else {
		d := model.Mode.ElementsAs(ctx, &modes, false)
		diags.Append(d...)
		if diags.HasError() {
			return fmt.Errorf("invalid mode list")
		}
	}

	return r.client.UpdateRateLimitSettings(model.Domain.ValueString(), client.RateLimitSettings{Mode: modes})
}

func (r *RateLimitSettingsResource) readRateLimitSettings(ctx context.Context, state *RateLimitSettingsResourceModel) error {
	rl, err := r.client.GetRateLimit(state.Domain.ValueString())
	if err != nil {
		return err
	}

	// Preserve nil vs empty slice where possible.
	if rl.Config.Mode == nil {
		state.Mode = types.ListNull(types.StringType)
	} else {
		values := make([]attr.Value, len(rl.Config.Mode))
		for i, m := range rl.Config.Mode {
			values[i] = types.StringValue(m)
		}
		state.Mode = types.ListValueMust(types.StringType, values)
	}

	return nil
}
