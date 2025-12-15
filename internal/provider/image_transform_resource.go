package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &ImageTransformResource{}
var _ resource.ResourceWithConfigure = &ImageTransformResource{}
var _ resource.ResourceWithImportState = &ImageTransformResource{}

func NewImageTransformResource() resource.Resource {
	return &ImageTransformResource{}
}

type ImageTransformResource struct {
	client *client.Client
}

type ImageTransformResourceModel struct {
	ID         types.String         `tfsdk:"id"`
	Domain     types.String         `tfsdk:"domain"`
	UUID       types.String         `tfsdk:"uuid"`
	Name       types.String         `tfsdk:"name"`
	ConfigJSON JSONNormalizedValue `tfsdk:"config_json"`
}

func (r *ImageTransformResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_transform"
}

func (r *ImageTransformResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an image transform preset for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Image transform identifier (domain/uuid).",
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
				Description: "Preset UUID (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Preset name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config_json": schema.StringAttribute{
				Description: "Transformation configuration as JSON string (e.g. '{\"width\": 800}'). Server-side defaults are ignored for drift detection.",
				Required:    true,
				CustomType:  JSONNormalizedType{},
			},
		},
	}
}

func (r *ImageTransformResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ImageTransformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ImageTransformResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse config JSON
	var config map[string]any
	if err := json.Unmarshal([]byte(plan.ConfigJSON.ValueString()), &config); err != nil {
		resp.Diagnostics.AddError(
			"Invalid config JSON",
			"Could not parse config_json: "+err.Error(),
		)
		return
	}

	// Create preset
	preset := client.ImageTransformPresetCreate{
		Name:   plan.Name.ValueString(),
		Config: config,
	}

	result, err := r.client.CreateImageTransformPreset(plan.Domain.ValueString(), preset)
	if err != nil {
		if client.IsConflictError(err) || isAlreadyExistsError(err) {
			// Try to find the existing preset to get its UUID
			existingUUID := "<uuid>"
			if presets, listErr := r.client.ListImageTransformPresets(plan.Domain.ValueString()); listErr == nil {
				for _, p := range presets {
					if p.Name == plan.Name.ValueString() {
						existingUUID = p.UUID
						break
					}
				}
			}
			resp.Diagnostics.AddError(
				"Image Transform Preset Already Exists",
				fmt.Sprintf("An image transform preset with name %q already exists for domain %q. To manage it with Terraform, add an import block:\n\n"+
					"  import {\n"+
					"    to = peakhour_image_transform.example\n"+
					"    id = \"%s/%s\"\n"+
					"  }\n\n"+
					"Then run: terraform apply",
					plan.Name.ValueString(), plan.Domain.ValueString(), plan.Domain.ValueString(), existingUUID),
			)
		} else {
			resp.Diagnostics.AddError(
				"Error creating image transform preset",
				"Could not create preset for domain "+plan.Domain.ValueString()+": "+err.Error(),
			)
		}
		return
	}

	// Note: Use peakhour_image_transform_commit resource to commit changes

	// Set computed values
	plan.UUID = types.StringValue(result.UUID)
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + result.UUID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ImageTransformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ImageTransformResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get preset from API
	preset, err := r.client.GetImageTransformPreset(state.Domain.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading image transform preset",
			"Could not read preset "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	state.Name = types.StringValue(preset.Name)
	state.ID = types.StringValue(state.Domain.ValueString() + "/" + state.UUID.ValueString())

	// Always set config_json from API - semantic equality handles drift detection
	configJSON, err := json.Marshal(preset.Config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error serializing config",
			"Could not serialize config to JSON: "+err.Error(),
		)
		return
	}
	state.ConfigJSON = NewJSONNormalizedValue(string(configJSON))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ImageTransformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ImageTransformResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse config JSON
	var config map[string]any
	if err := json.Unmarshal([]byte(plan.ConfigJSON.ValueString()), &config); err != nil {
		resp.Diagnostics.AddError(
			"Invalid config JSON",
			"Could not parse config_json: "+err.Error(),
		)
		return
	}

	// Update preset
	update := client.ImageTransformPresetUpdate{
		Config: config,
	}

	// Wait, UpdateImageTransformPreset takes Update struct which only has Config?
	// If Name can be updated, the client and API should support it.
	// Looking at client/types.go:
	// type ImageTransformPresetUpdate struct {
	// 	Config map[string]any `json:"config"`
	// }
	// It seems Name cannot be updated via this struct?
	// If the name changed in the plan, we might need to recreate the resource or check if there is a way to update name.
	// However, Schema defines Name as Required but not RequiresReplace?
	// If Name cannot be updated, I should mark it as RequiresReplace.
	// Let's check `internal/client/transform.go`.

	// Re-checking types: yes, ImageTransformPresetUpdate only has Config.
	// Re-checking `docs/spec/peakhour-api-v1.json`:
	// It's likely Name is immutable or updated via a different way, or the struct definition in client is incomplete?
	// Assuming client struct is correct -> Name update is impossible -> RequiresReplace.

	// But wait, if I can't update name, I should check if plan.Name differs from state.Name (if I had access to state).
	// But in `Update` I only see plan.
	// Ah, I should set RequiresReplace on Name in Schema if it's immutable.

	// Let's assume Name is immutable for now based on struct.

	err := r.client.UpdateImageTransformPreset(plan.Domain.ValueString(), plan.UUID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating image transform preset",
			"Could not update preset "+plan.UUID.ValueString()+" for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Note: Use peakhour_image_transform_commit resource to commit changes

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ImageTransformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ImageTransformResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete preset via API
	err := r.client.DeleteImageTransformPreset(state.Domain.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting image transform preset",
			"Could not delete preset "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Note: Use peakhour_image_transform_commit resource to commit changes
}

func (r *ImageTransformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/uuid
	parts, err := parseCompositeID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/uuid', got %q: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
