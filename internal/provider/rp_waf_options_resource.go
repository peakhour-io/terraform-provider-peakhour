package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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

var _ resource.Resource = &RPWAFOptionsResource{}
var _ resource.ResourceWithConfigure = &RPWAFOptionsResource{}
var _ resource.ResourceWithImportState = &RPWAFOptionsResource{}

func NewRPWAFOptionsResource() resource.Resource {
	return &RPWAFOptionsResource{}
}

type RPWAFOptionsResource struct {
	client *client.Client
}

type RPWAFOptionsResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Domain                      types.String `tfsdk:"domain"`
	WAFMode                     types.String `tfsdk:"waf_mode"`
	WAFRuleset                  types.String `tfsdk:"waf_ruleset"`
	WAFSetExposedPasswordHeader types.Bool   `tfsdk:"waf_set_exposed_password_header"`

	WAFOwaspVersion   types.String         `tfsdk:"waf_owasp_version"`
	ExcludedRulesJSON jsontypes.Normalized `tfsdk:"excluded_rules_json"`
	ExcludedFilesJSON jsontypes.Normalized `tfsdk:"excluded_files_json"`
}

func (r *RPWAFOptionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_waf_options"
}

func (r *RPWAFOptionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages reverse proxy WAF options for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "WAF options identifier (domain/rp_waf_options).",
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
			"waf_mode": schema.StringAttribute{
				Description: "WAF mode (spec enum: disabled, enabled, warn). Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"waf_ruleset": schema.StringAttribute{
				Description: "WAF rule set name (spec enum: atomic, owaspv33, owasp_virtual_patches, virtual_patches, peakhour). Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"waf_set_exposed_password_header": schema.BoolAttribute{
				Description: "Whether to set an exposed password header. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"waf_owasp_version": schema.StringAttribute{
				Description: "OWASP ruleset version (computed; configured via OWASP settings).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"excluded_rules_json": schema.StringAttribute{
				Description: "Computed JSON array of excluded rules (disabled rules).",
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"excluded_files_json": schema.StringAttribute{
				Description: "Computed JSON array of excluded rule groups (disabled files).",
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPWAFOptionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPWAFOptionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPWAFOptionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating WAF options", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_waf_options")
	if err := r.read(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF options after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFOptionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPWAFOptionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.read(ctx, &state, &resp.Diagnostics); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WAF options", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_waf_options")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPWAFOptionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPWAFOptionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating WAF options", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_waf_options")
	if err := r.read(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF options after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFOptionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPWAFOptionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"waf_mode":                        nil,
		"waf_ruleset":                     nil,
		"waf_set_exposed_password_header": nil,
	}

	if err := r.client.UpdateRPWAFOptions(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting WAF options", err.Error())
		return
	}
}

func (r *RPWAFOptionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPWAFOptionsResource) apply(ctx context.Context, model *RPWAFOptionsResourceModel, diags *diag.Diagnostics) error {
	update := map[string]any{}

	if !model.WAFMode.IsUnknown() {
		if model.WAFMode.IsNull() {
			update["waf_mode"] = nil
		} else {
			update["waf_mode"] = model.WAFMode.ValueString()
		}
	}

	if !model.WAFRuleset.IsUnknown() {
		if model.WAFRuleset.IsNull() {
			update["waf_ruleset"] = nil
		} else {
			update["waf_ruleset"] = model.WAFRuleset.ValueString()
		}
	}

	if !model.WAFSetExposedPasswordHeader.IsUnknown() {
		if model.WAFSetExposedPasswordHeader.IsNull() {
			update["waf_set_exposed_password_header"] = nil
		} else {
			update["waf_set_exposed_password_header"] = model.WAFSetExposedPasswordHeader.ValueBool()
		}
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPWAFOptions(model.Domain.ValueString(), update)
}

func (r *RPWAFOptionsResource) read(ctx context.Context, model *RPWAFOptionsResourceModel, diags *diag.Diagnostics) error {
	options, err := r.client.GetRPWAFOptions(model.Domain.ValueString())
	if err != nil {
		return err
	}

	if options.WAFMode == nil {
		model.WAFMode = types.StringNull()
	} else {
		model.WAFMode = types.StringValue(*options.WAFMode)
	}

	if options.WAFRuleset == nil {
		model.WAFRuleset = types.StringNull()
	} else {
		model.WAFRuleset = types.StringValue(*options.WAFRuleset)
	}

	if options.WAFSetExposedPasswordHeader == nil {
		model.WAFSetExposedPasswordHeader = types.BoolNull()
	} else {
		model.WAFSetExposedPasswordHeader = types.BoolValue(*options.WAFSetExposedPasswordHeader)
	}

	if options.WAFOwaspVersion == nil {
		model.WAFOwaspVersion = types.StringNull()
	} else {
		model.WAFOwaspVersion = types.StringValue(*options.WAFOwaspVersion)
	}

	// Always set JSON fields from API - semantic equality handles drift detection
	excludedRulesRaw, err := json.Marshal(options.WAFExcludedRules)
	if err != nil {
		diags.AddError("Error marshalling excluded rules", err.Error())
		return nil
	}
	model.ExcludedRulesJSON = jsontypes.NewNormalizedValue(string(excludedRulesRaw))

	excludedFilesRaw, err := json.Marshal(options.WAFExcludedFiles)
	if err != nil {
		diags.AddError("Error marshalling excluded files", err.Error())
		return nil
	}
	model.ExcludedFilesJSON = jsontypes.NewNormalizedValue(string(excludedFilesRaw))

	return nil
}
