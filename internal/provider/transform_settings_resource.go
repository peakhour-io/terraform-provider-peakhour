package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &TransformSettingsResource{}
var _ resource.ResourceWithConfigure = &TransformSettingsResource{}
var _ resource.ResourceWithImportState = &TransformSettingsResource{}

func NewTransformSettingsResource() resource.Resource {
	return &TransformSettingsResource{}
}

type TransformSettingsResource struct {
	client *client.Client
}

type TransformSettingsResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Domain                      types.String `tfsdk:"domain"`
	TransformHTML               types.Bool   `tfsdk:"transform_html"`
	TransformBeacon             types.Bool   `tfsdk:"transform_beacon"`
	TransformLazySizes          types.Bool   `tfsdk:"transform_lazy_sizes"`
	TransformMixedContent       types.Bool   `tfsdk:"transform_mixed_content"`
	TransformImgDimsToQueryArgs types.Bool   `tfsdk:"transform_img_dims_to_query_args"`
	TransformImageQuality       types.Int64  `tfsdk:"transform_image_quality"`
	TransformImageFormat        types.Bool   `tfsdk:"transform_image_format"`
	TransformImageOptimise      types.Bool   `tfsdk:"transform_image_optimise"`
	TransformImageAPI           types.Bool   `tfsdk:"transform_image_api"`
	TransformHTTPHeaderValue    types.String `tfsdk:"transform_http_header_value"`
	TransformESI                types.Bool   `tfsdk:"transform_esi"`
	TransformRewriteDomains     types.List   `tfsdk:"transform_rewrite_domains"`
}

func (r *TransformSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform_settings"
}

func (r *TransformSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages transform settings (HTML/image processing) for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Transform settings identifier.",
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
			"transform_html": schema.BoolAttribute{
				Description: "Enable HTML transformation.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_beacon": schema.BoolAttribute{
				Description: "Enable beacon injection.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_lazy_sizes": schema.BoolAttribute{
				Description: "Enable lazy loading for images.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_mixed_content": schema.BoolAttribute{
				Description: "Fix mixed content (HTTP/HTTPS).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_img_dims_to_query_args": schema.BoolAttribute{
				Description: "Convert image dimensions to query arguments.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_image_quality": schema.Int64Attribute{
				Description: "Image quality (1-100).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(85),
			},
			"transform_image_format": schema.BoolAttribute{
				Description: "Enable automatic image format conversion.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_image_optimise": schema.BoolAttribute{
				Description: "Enable image optimization.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_image_api": schema.BoolAttribute{
				Description: "Enable image transformation API.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_http_header_value": schema.StringAttribute{
				Description: "HTTP header value for transforms.",
				Optional:    true,
			},
			"transform_esi": schema.BoolAttribute{
				Description: "Enable Edge Side Includes (ESI).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transform_rewrite_domains": schema.ListAttribute{
				Description: "Domains to rewrite in transformed content.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *TransformSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TransformSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransformSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build settings
	settings := r.buildSettingsFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update settings via API
	err := r.client.UpdateTransformSettings(plan.Domain.ValueString(), settings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating transform settings",
			"Could not create settings for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set ID
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/transforms")

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *TransformSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransformSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get settings from API
	settings, err := r.client.GetTransformSettings(state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading transform settings",
			"Could not read settings for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state
	r.mapSettingsToModel(ctx, settings, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *TransformSettingsResource) mapSettingsToModel(ctx context.Context, settings *client.TransformSettings, state *TransformSettingsResourceModel) {
	if settings.TransformHTML != nil {
		state.TransformHTML = types.BoolValue(*settings.TransformHTML)
	} else {
		state.TransformHTML = types.BoolNull()
	}
	if settings.TransformBeacon != nil {
		state.TransformBeacon = types.BoolValue(*settings.TransformBeacon)
	} else {
		state.TransformBeacon = types.BoolNull()
	}
	if settings.TransformLazySizes != nil {
		state.TransformLazySizes = types.BoolValue(*settings.TransformLazySizes)
	} else {
		state.TransformLazySizes = types.BoolNull()
	}
	if settings.TransformMixedContent != nil {
		state.TransformMixedContent = types.BoolValue(*settings.TransformMixedContent)
	} else {
		state.TransformMixedContent = types.BoolNull()
	}
	if settings.TransformImgDimsToQueryArgs != nil {
		state.TransformImgDimsToQueryArgs = types.BoolValue(*settings.TransformImgDimsToQueryArgs)
	} else {
		state.TransformImgDimsToQueryArgs = types.BoolNull()
	}
	if settings.TransformImageQuality != nil {
		state.TransformImageQuality = types.Int64Value(int64(*settings.TransformImageQuality))
	} else {
		state.TransformImageQuality = types.Int64Null()
	}
	if settings.TransformImageFormat != nil {
		state.TransformImageFormat = types.BoolValue(*settings.TransformImageFormat)
	} else {
		state.TransformImageFormat = types.BoolNull()
	}
	if settings.TransformImageOptimise != nil {
		state.TransformImageOptimise = types.BoolValue(*settings.TransformImageOptimise)
	} else {
		state.TransformImageOptimise = types.BoolNull()
	}
	if settings.TransformImageAPI != nil {
		state.TransformImageAPI = types.BoolValue(*settings.TransformImageAPI)
	} else {
		state.TransformImageAPI = types.BoolNull()
	}
	if settings.TransformHTTPHeaderValue != nil {
		state.TransformHTTPHeaderValue = types.StringValue(*settings.TransformHTTPHeaderValue)
	} else {
		state.TransformHTTPHeaderValue = types.StringNull()
	}
	if settings.TransformESI != nil {
		state.TransformESI = types.BoolValue(*settings.TransformESI)
	} else {
		state.TransformESI = types.BoolNull()
	}

	if len(settings.TransformRewriteDomains) > 0 {
		domains := make([]attr.Value, len(settings.TransformRewriteDomains))
		for i, domain := range settings.TransformRewriteDomains {
			domains[i] = types.StringValue(domain)
		}
		state.TransformRewriteDomains = types.ListValueMust(types.StringType, domains)
	} else {
		state.TransformRewriteDomains = types.ListNull(types.StringType)
	}
}

func (r *TransformSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TransformSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build settings
	settings := r.buildSettingsFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update settings via API
	err := r.client.UpdateTransformSettings(plan.Domain.ValueString(), settings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating transform settings",
			"Could not update settings for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *TransformSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransformSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Send empty settings to reset
	if err := r.resetSettings(ctx, state.Domain.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting transform settings",
			"Could not reset settings for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *TransformSettingsResource) resetSettings(ctx context.Context, domain string) error {
	settings := client.TransformSettings{}
	return r.client.UpdateTransformSettings(domain, settings)
}

func (r *TransformSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *TransformSettingsResource) buildSettingsFromModel(ctx context.Context, model *TransformSettingsResourceModel, diags *diag.Diagnostics) client.TransformSettings {
	settings := client.TransformSettings{}

	if !model.TransformHTML.IsNull() {
		v := model.TransformHTML.ValueBool()
		settings.TransformHTML = &v
	}
	if !model.TransformBeacon.IsNull() {
		v := model.TransformBeacon.ValueBool()
		settings.TransformBeacon = &v
	}
	if !model.TransformLazySizes.IsNull() {
		v := model.TransformLazySizes.ValueBool()
		settings.TransformLazySizes = &v
	}
	if !model.TransformMixedContent.IsNull() {
		v := model.TransformMixedContent.ValueBool()
		settings.TransformMixedContent = &v
	}
	if !model.TransformImgDimsToQueryArgs.IsNull() {
		v := model.TransformImgDimsToQueryArgs.ValueBool()
		settings.TransformImgDimsToQueryArgs = &v
	}
	if !model.TransformImageQuality.IsNull() {
		v := int(model.TransformImageQuality.ValueInt64())
		settings.TransformImageQuality = &v
	}
	if !model.TransformImageFormat.IsNull() {
		v := model.TransformImageFormat.ValueBool()
		settings.TransformImageFormat = &v
	}
	if !model.TransformImageOptimise.IsNull() {
		v := model.TransformImageOptimise.ValueBool()
		settings.TransformImageOptimise = &v
	}
	if !model.TransformImageAPI.IsNull() {
		v := model.TransformImageAPI.ValueBool()
		settings.TransformImageAPI = &v
	}
	if !model.TransformHTTPHeaderValue.IsNull() {
		v := model.TransformHTTPHeaderValue.ValueString()
		settings.TransformHTTPHeaderValue = &v
	}
	if !model.TransformESI.IsNull() {
		v := model.TransformESI.ValueBool()
		settings.TransformESI = &v
	}

	if !model.TransformRewriteDomains.IsNull() {
		var domains []string
		d := model.TransformRewriteDomains.ElementsAs(ctx, &domains, false)
		diags.Append(d...)
		if !diags.HasError() {
			settings.TransformRewriteDomains = domains
		}
	}

	return settings
}
