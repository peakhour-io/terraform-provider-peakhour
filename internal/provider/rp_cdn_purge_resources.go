package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPCDNPurgeResourcesResource{}
var _ resource.ResourceWithConfigure = &RPCDNPurgeResourcesResource{}

var _ resource.Resource = &RPCDNPurgeWildcardResource{}
var _ resource.ResourceWithConfigure = &RPCDNPurgeWildcardResource{}

var _ resource.Resource = &RPCDNPurgeTagsResource{}
var _ resource.ResourceWithConfigure = &RPCDNPurgeTagsResource{}

func NewRPCDNPurgeResourcesResource() resource.Resource {
	return &RPCDNPurgeResourcesResource{}
}

func NewRPCDNPurgeWildcardResource() resource.Resource {
	return &RPCDNPurgeWildcardResource{}
}

func NewRPCDNPurgeTagsResource() resource.Resource {
	return &RPCDNPurgeTagsResource{}
}

type RPCDNPurgeResourcesResource struct {
	client *client.Client
}

type RPCDNPurgeResourcesResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	RunID  types.String `tfsdk:"run_id"`
	Paths  types.List   `tfsdk:"paths"`
	Soft   types.Bool   `tfsdk:"soft"`
}

func (r *RPCDNPurgeResourcesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_cdn_purge_resources"
}

func (r *RPCDNPurgeResourcesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers a reverse proxy CDN purge for specific resources (action). This is not stable desired-state; changing inputs will re-run the purge.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Purge identifier (domain/rp_cdn_purge_resources/run_id).",
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
			"run_id": schema.StringAttribute{
				Description: "An arbitrary identifier to force a new purge (e.g. timestamp or ticket number).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"paths": schema.ListAttribute{
				Description: "Paths to purge. Set to null to purge all resources.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"soft": schema.BoolAttribute{
				Description: "Whether to use soft purge semantics. Set to null to use API default.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RPCDNPurgeResourcesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPCDNPurgeResourcesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPCDNPurgeResourcesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	purge := buildPurgeFromModel(ctx, plan.Paths, plan.Soft, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.FlushRPCDNResources(plan.Domain.ValueString(), purge); err != nil {
		resp.Diagnostics.AddError("Error purging CDN resources", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/rp_cdn_purge_resources/%s", plan.Domain.ValueString(), plan.RunID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNPurgeResourcesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPCDNPurgeResourcesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPCDNPurgeResourcesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPCDNPurgeResourcesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	purge := buildPurgeFromModel(ctx, plan.Paths, plan.Soft, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.FlushRPCDNResources(plan.Domain.ValueString(), purge); err != nil {
		resp.Diagnostics.AddError("Error purging CDN resources", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/rp_cdn_purge_resources/%s", plan.Domain.ValueString(), plan.RunID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNPurgeResourcesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// no-op: this resource models an action
}

type RPCDNPurgeWildcardResource struct {
	client *client.Client
}

type RPCDNPurgeWildcardResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	RunID  types.String `tfsdk:"run_id"`
	Paths  types.List   `tfsdk:"paths"`
	Soft   types.Bool   `tfsdk:"soft"`
}

func (r *RPCDNPurgeWildcardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_cdn_purge_wildcard"
}

func (r *RPCDNPurgeWildcardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers a reverse proxy CDN purge for wildcard paths (action). This is not stable desired-state; changing inputs will re-run the purge.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Purge identifier (domain/rp_cdn_purge_wildcard/run_id).",
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
			"run_id": schema.StringAttribute{
				Description: "An arbitrary identifier to force a new purge (e.g. timestamp or ticket number).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"paths": schema.ListAttribute{
				Description: "Wildcard paths to purge. Set to null to purge all wildcard resources.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"soft": schema.BoolAttribute{
				Description: "Whether to use soft purge semantics. Set to null to use API default.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RPCDNPurgeWildcardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPCDNPurgeWildcardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPCDNPurgeWildcardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	purge := buildPurgeFromModel(ctx, plan.Paths, plan.Soft, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.FlushRPCDNWildcard(plan.Domain.ValueString(), purge); err != nil {
		resp.Diagnostics.AddError("Error purging CDN wildcard paths", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/rp_cdn_purge_wildcard/%s", plan.Domain.ValueString(), plan.RunID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNPurgeWildcardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPCDNPurgeWildcardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPCDNPurgeWildcardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPCDNPurgeWildcardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	purge := buildPurgeFromModel(ctx, plan.Paths, plan.Soft, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.FlushRPCDNWildcard(plan.Domain.ValueString(), purge); err != nil {
		resp.Diagnostics.AddError("Error purging CDN wildcard paths", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/rp_cdn_purge_wildcard/%s", plan.Domain.ValueString(), plan.RunID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNPurgeWildcardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// no-op: this resource models an action
}

type RPCDNPurgeTagsResource struct {
	client *client.Client
}

type RPCDNPurgeTagsResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
	RunID  types.String `tfsdk:"run_id"`
	Tags   types.List   `tfsdk:"tags"`
	Soft   types.Bool   `tfsdk:"soft"`
}

func (r *RPCDNPurgeTagsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_cdn_purge_tags"
}

func (r *RPCDNPurgeTagsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers a reverse proxy CDN purge by cache tag (action). This is not stable desired-state; changing inputs will re-run the purge.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Purge identifier (domain/rp_cdn_purge_tags/run_id).",
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
			"run_id": schema.StringAttribute{
				Description: "An arbitrary identifier to force a new purge (e.g. timestamp or ticket number).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.ListAttribute{
				Description: "Cache tags to purge.",
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"soft": schema.BoolAttribute{
				Description: "Whether to use soft purge semantics. Set to null to use API default.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RPCDNPurgeTagsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPCDNPurgeTagsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPCDNPurgeTagsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags := buildStringList(ctx, plan.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(tags) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("tags"), "Missing tags", "At least one tag is required.")
		return
	}

	purge := client.PurgeTags{
		Tags: tags,
		Soft: buildOptionalBool(plan.Soft),
	}

	if err := r.client.FlushRPCDNTags(plan.Domain.ValueString(), purge); err != nil {
		resp.Diagnostics.AddError("Error purging CDN tags", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/rp_cdn_purge_tags/%s", plan.Domain.ValueString(), plan.RunID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNPurgeTagsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPCDNPurgeTagsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPCDNPurgeTagsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPCDNPurgeTagsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags := buildStringList(ctx, plan.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(tags) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("tags"), "Missing tags", "At least one tag is required.")
		return
	}

	purge := client.PurgeTags{
		Tags: tags,
		Soft: buildOptionalBool(plan.Soft),
	}

	if err := r.client.FlushRPCDNTags(plan.Domain.ValueString(), purge); err != nil {
		resp.Diagnostics.AddError("Error purging CDN tags", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/rp_cdn_purge_tags/%s", plan.Domain.ValueString(), plan.RunID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPCDNPurgeTagsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// no-op: this resource models an action
}

func buildPurgeFromModel(ctx context.Context, paths types.List, soft types.Bool, diags *diag.Diagnostics) client.Purge {
	purge := client.Purge{
		Soft: buildOptionalBool(soft),
	}

	if paths.IsNull() || paths.IsUnknown() {
		return purge
	}

	list := buildStringList(ctx, paths, diags)
	if diags.HasError() {
		return purge
	}
	purge.Paths = list
	return purge
}

func buildStringList(ctx context.Context, input types.List, diags *diag.Diagnostics) []string {
	if input.IsNull() || input.IsUnknown() {
		return nil
	}

	var out []string
	diags.Append(input.ElementsAs(ctx, &out, false)...)
	if diags.HasError() {
		return nil
	}
	return out
}

func buildOptionalBool(input types.Bool) *bool {
	if input.IsNull() || input.IsUnknown() {
		return nil
	}
	v := input.ValueBool()
	return &v
}
