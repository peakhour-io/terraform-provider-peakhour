package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPWAFOWASPSettingsResource{}
var _ resource.ResourceWithConfigure = &RPWAFOWASPSettingsResource{}
var _ resource.ResourceWithImportState = &RPWAFOWASPSettingsResource{}

func NewRPWAFOWASPSettingsResource() resource.Resource {
	return &RPWAFOWASPSettingsResource{}
}

type RPWAFOWASPSettingsResource struct {
	client *client.Client
}

type RPWAFOWASPSettingsResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Domain       types.String `tfsdk:"domain"`
	SettingsJSON types.String `tfsdk:"settings_json"`
}

func (r *RPWAFOWASPSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_waf_owasp_settings"
}

func (r *RPWAFOWASPSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages reverse proxy WAF OWASP settings for a domain. The schema is represented as JSON.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "WAF OWASP settings identifier (domain/rp_waf_owasp_settings).",
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
			"settings_json": schema.StringAttribute{
				Description: "OWASP settings as a JSON object. This is PATCHed to the API; omitted keys are left unchanged, and explicit nulls clear values.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPWAFOWASPSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPWAFOWASPSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPWAFOWASPSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating WAF OWASP settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_waf_owasp_settings")
	if err := r.read(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF OWASP settings after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFOWASPSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPWAFOWASPSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.read(ctx, &state, &resp.Diagnostics); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WAF OWASP settings", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_waf_owasp_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPWAFOWASPSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPWAFOWASPSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating WAF OWASP settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_waf_owasp_settings")
	if err := r.read(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF OWASP settings after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFOWASPSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPWAFOWASPSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"peakhour_settings": nil,
		"initialization":    nil,
		"protocol":          nil,
		"methods":           nil,
	}

	if err := r.client.UpdateRPWAFOWASPSettings(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting WAF OWASP settings", err.Error())
		return
	}
}

func (r *RPWAFOWASPSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPWAFOWASPSettingsResource) apply(ctx context.Context, model *RPWAFOWASPSettingsResourceModel, diags *diag.Diagnostics) error {
	if model.SettingsJSON.IsUnknown() || model.SettingsJSON.IsNull() {
		return nil
	}

	normalized, err := normalizeJSON(model.SettingsJSON.ValueString())
	if err != nil {
		return fmt.Errorf("invalid settings_json: %w", err)
	}

	var update map[string]any
	if err := json.Unmarshal([]byte(normalized), &update); err != nil {
		return fmt.Errorf("invalid settings_json: %w", err)
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPWAFOWASPSettings(model.Domain.ValueString(), update)
}

func (r *RPWAFOWASPSettingsResource) read(ctx context.Context, model *RPWAFOWASPSettingsResourceModel, diags *diag.Diagnostics) error {
	settings, err := r.client.GetRPWAFOWASPSettings(model.Domain.ValueString())
	if err != nil {
		return err
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		diags.AddError("Error marshalling OWASP settings", err.Error())
		return nil
	}

	normalized, err := normalizeJSON(string(raw))
	if err != nil {
		diags.AddError("Error normalizing OWASP settings JSON", err.Error())
		return nil
	}

	model.SettingsJSON = types.StringValue(normalized)
	return nil
}
