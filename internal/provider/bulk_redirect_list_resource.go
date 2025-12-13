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

var _ resource.Resource = &BulkRedirectListResource{}
var _ resource.ResourceWithConfigure = &BulkRedirectListResource{}
var _ resource.ResourceWithImportState = &BulkRedirectListResource{}

func NewBulkRedirectListResource() resource.Resource {
	return &BulkRedirectListResource{}
}

type BulkRedirectListResource struct {
	client *client.Client
}

type BulkRedirectListResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Domain       types.String `tfsdk:"domain"`
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	EntriesCount types.Int64  `tfsdk:"entries_count"`
}

func (r *BulkRedirectListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bulk_redirect_list"
}

func (r *BulkRedirectListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bulk redirect list for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Bulk redirect list identifier (domain/bulk_redirects/uuid).",
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
				Description: "Bulk redirect list UUID (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Bulk redirect list name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the redirect list.",
				Optional:    true,
			},
			"entries_count": schema.Int64Attribute{
				Description: "Number of entries in the list.",
				Computed:    true,
			},
		},
	}
}

func (r *BulkRedirectListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BulkRedirectListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BulkRedirectListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	create := client.BulkRedirectListCreate{Name: &name}
	if !plan.Description.IsNull() {
		desc := plan.Description.ValueString()
		create.Description = &desc
	}

	result, err := r.client.CreateBulkRedirectList(plan.Domain.ValueString(), create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating bulk redirect list",
			"Could not create bulk redirect list for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	list, err := r.client.GetBulkRedirectList(plan.Domain.ValueString(), result.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bulk redirect list",
			"Could not read created bulk redirect list for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.UUID = types.StringValue(list.UUID)
	plan.ID = types.StringValue(fmt.Sprintf("%s/bulk_redirects/%s", plan.Domain.ValueString(), list.UUID))
	plan.EntriesCount = types.Int64Value(int64(list.EntriesCount))

	if list.Name != nil {
		plan.Name = types.StringValue(*list.Name)
	}
	if list.Description != nil {
		plan.Description = types.StringValue(*list.Description)
	} else {
		plan.Description = types.StringNull()
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *BulkRedirectListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BulkRedirectListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := r.client.GetBulkRedirectList(state.Domain.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading bulk redirect list",
			"Could not read bulk redirect list "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	state.UUID = types.StringValue(list.UUID)
	state.ID = types.StringValue(fmt.Sprintf("%s/bulk_redirects/%s", state.Domain.ValueString(), list.UUID))
	state.EntriesCount = types.Int64Value(int64(list.EntriesCount))

	if list.Name != nil {
		state.Name = types.StringValue(*list.Name)
	} else {
		state.Name = types.StringNull()
	}
	if list.Description != nil {
		state.Description = types.StringValue(*list.Description)
	} else {
		state.Description = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *BulkRedirectListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BulkRedirectListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	update := client.BulkRedirectListUpdate{Name: &name}
	if !plan.Description.IsNull() {
		desc := plan.Description.ValueString()
		update.Description = &desc
	}

	if err := r.client.UpdateBulkRedirectList(plan.Domain.ValueString(), plan.UUID.ValueString(), update); err != nil {
		resp.Diagnostics.AddError(
			"Error updating bulk redirect list",
			"Could not update bulk redirect list "+plan.UUID.ValueString()+" for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *BulkRedirectListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BulkRedirectListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteBulkRedirectList(state.Domain.ValueString(), state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting bulk redirect list",
			"Could not delete bulk redirect list "+state.UUID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *BulkRedirectListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/bulk_redirects/uuid
	parts, err := parseCompositeID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/bulk_redirects/uuid', got %q: %s", req.ID, err),
		)
		return
	}
	if parts[1] != "bulk_redirects" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/bulk_redirects/uuid', got %q: middle segment must be 'bulk_redirects'", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
