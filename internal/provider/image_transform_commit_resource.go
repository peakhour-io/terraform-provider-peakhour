package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &ImageTransformCommitResource{}
var _ resource.ResourceWithConfigure = &ImageTransformCommitResource{}

func NewImageTransformCommitResource() resource.Resource {
	return &ImageTransformCommitResource{}
}

type ImageTransformCommitResource struct {
	client *client.Client
}

type ImageTransformCommitResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Domain   types.String `tfsdk:"domain"`
	Triggers types.Map    `tfsdk:"triggers"`
}

func (r *ImageTransformCommitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_transform_commit"
}

func (r *ImageTransformCommitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Commits image transform changes for a domain. Use triggers to control when commits happen.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Commit resource identifier.",
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
			"triggers": schema.MapAttribute{
				Description: "Map of values that trigger a commit when changed. Use image transform IDs to commit after changes.",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *ImageTransformCommitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ImageTransformCommitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ImageTransformCommitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Commit changes
	if err := r.client.CommitImageTransforms(plan.Domain.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error committing image transforms",
			"Could not commit changes for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/image-transform-commit")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ImageTransformCommitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ImageTransformCommitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing to read - this is a trigger resource
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ImageTransformCommitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ImageTransformCommitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Commit changes when triggers change
	if err := r.client.CommitImageTransforms(plan.Domain.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error committing image transforms",
			"Could not commit changes for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ImageTransformCommitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to delete - this is just a trigger resource
}
