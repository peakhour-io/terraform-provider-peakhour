package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &BulkRedirectEntryResource{}
var _ resource.ResourceWithConfigure = &BulkRedirectEntryResource{}
var _ resource.ResourceWithImportState = &BulkRedirectEntryResource{}

func NewBulkRedirectEntryResource() resource.Resource {
	return &BulkRedirectEntryResource{}
}

type BulkRedirectEntryResource struct {
	client *client.Client
}

type BulkRedirectEntryResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Domain              types.String `tfsdk:"domain"`
	BulkRedirectUUID    types.String `tfsdk:"bulk_redirect_uuid"`
	EntryID             types.String `tfsdk:"entry_id"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	PreserveQueryString types.Bool   `tfsdk:"preserve_query_string"`
	SourceDomain        types.String `tfsdk:"source_domain"`
	SourcePath          types.String `tfsdk:"source_path"`
	SourceScheme        types.String `tfsdk:"source_scheme"`
	StatusCode          types.Int64  `tfsdk:"status_code"`
	TargetURL           types.String `tfsdk:"target_url"`
}

func (r *BulkRedirectEntryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bulk_redirect_entry"
}

func (r *BulkRedirectEntryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an entry in a bulk redirect list for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Bulk redirect entry identifier (domain/bulk_redirects/uuid/entries/entry_id).",
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
			"bulk_redirect_uuid": schema.StringAttribute{
				Description: "Bulk redirect list UUID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"entry_id": schema.StringAttribute{
				Description: "Entry identifier (computed after creation).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable/disable this redirect entry.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"preserve_query_string": schema.BoolAttribute{
				Description: "Preserve query string from source URL.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"source_scheme": schema.StringAttribute{
				Description: "Scheme to match (e.g. http or https).",
				Optional:    true,
			},
			"source_domain": schema.StringAttribute{
				Description: "Domain to match (e.g. www.example.com).",
				Optional:    true,
			},
			"source_path": schema.StringAttribute{
				Description: "Path to match for redirect (e.g. /old-page).",
				Required:    true,
			},
			"target_url": schema.StringAttribute{
				Description: "Full URL to redirect to.",
				Required:    true,
			},
			"status_code": schema.Int64Attribute{
				Description: "HTTP status code for the redirect (301, 302, 307, 308).",
				Optional:    true,
			},
		},
	}
}

func (r *BulkRedirectEntryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BulkRedirectEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BulkRedirectEntryResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	create := r.buildEntryCreateFromModel(&plan)
	sourcePath := plan.SourcePath.ValueString()
	targetURL := plan.TargetURL.ValueString()
	create.SourcePath = &sourcePath
	create.TargetURL = &targetURL

	entry, err := r.client.CreateBulkRedirectEntry(plan.Domain.ValueString(), plan.BulkRedirectUUID.ValueString(), create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating bulk redirect entry",
			"Could not create bulk redirect entry for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.EntryID = types.StringValue(entry.ID.String())
	plan.ID = types.StringValue(fmt.Sprintf("%s/bulk_redirects/%s/entries/%s", plan.Domain.ValueString(), plan.BulkRedirectUUID.ValueString(), entry.ID.String()))

	r.updateModelFromEntry(&plan, entry)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *BulkRedirectEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BulkRedirectEntryResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, err := r.client.ListBulkRedirectEntries(state.Domain.ValueString(), state.BulkRedirectUUID.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading bulk redirect entry",
			"Could not list bulk redirect entries for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	var found *client.BulkRedirectEntry
	for i := range entries {
		if entries[i].ID.String() == state.EntryID.ValueString() {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/bulk_redirects/%s/entries/%s", state.Domain.ValueString(), state.BulkRedirectUUID.ValueString(), found.ID.String()))
	r.updateModelFromEntry(&state, found)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *BulkRedirectEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BulkRedirectEntryResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := r.buildEntryUpdateFromModel(&plan)

	if err := r.client.UpdateBulkRedirectEntry(plan.Domain.ValueString(), plan.BulkRedirectUUID.ValueString(), plan.EntryID.ValueString(), update); err != nil {
		resp.Diagnostics.AddError(
			"Error updating bulk redirect entry",
			"Could not update bulk redirect entry "+plan.EntryID.ValueString()+" for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *BulkRedirectEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BulkRedirectEntryResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteBulkRedirectEntry(state.Domain.ValueString(), state.BulkRedirectUUID.ValueString(), state.EntryID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting bulk redirect entry",
			"Could not delete bulk redirect entry "+state.EntryID.ValueString()+" for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *BulkRedirectEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain/bulk_redirects/uuid/entries/entry_id
	parts, err := parseCompositeID(req.ID, 5)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/bulk_redirects/uuid/entries/entry_id', got %q: %s", req.ID, err),
		)
		return
	}
	if parts[1] != "bulk_redirects" || parts[3] != "entries" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/bulk_redirects/uuid/entries/entry_id', got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bulk_redirect_uuid"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry_id"), parts[4])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *BulkRedirectEntryResource) buildEntryCreateFromModel(model *BulkRedirectEntryResourceModel) client.BulkRedirectEntryCreate {
	out := client.BulkRedirectEntryCreate{}

	if !model.Enabled.IsNull() {
		v := model.Enabled.ValueBool()
		out.Enabled = &v
	}
	if !model.PreserveQueryString.IsNull() {
		v := model.PreserveQueryString.ValueBool()
		out.PreserveQueryString = &v
	}
	if !model.SourceDomain.IsNull() {
		v := model.SourceDomain.ValueString()
		out.SourceDomain = &v
	}
	if !model.SourceScheme.IsNull() {
		v := model.SourceScheme.ValueString()
		out.SourceScheme = &v
	}
	if !model.StatusCode.IsNull() {
		v := client.RedirectStatusCode(int(model.StatusCode.ValueInt64()))
		out.StatusCode = &v
	}

	return out
}

func (r *BulkRedirectEntryResource) buildEntryUpdateFromModel(model *BulkRedirectEntryResourceModel) client.BulkRedirectEntryUpdate {
	out := client.BulkRedirectEntryUpdate{}

	if !model.Enabled.IsNull() {
		v := model.Enabled.ValueBool()
		out.Enabled = &v
	}
	if !model.PreserveQueryString.IsNull() {
		v := model.PreserveQueryString.ValueBool()
		out.PreserveQueryString = &v
	}
	if !model.SourceDomain.IsNull() {
		v := model.SourceDomain.ValueString()
		out.SourceDomain = &v
	}
	if !model.SourceScheme.IsNull() {
		v := model.SourceScheme.ValueString()
		out.SourceScheme = &v
	}

	sourcePath := model.SourcePath.ValueString()
	targetURL := model.TargetURL.ValueString()
	out.SourcePath = &sourcePath
	out.TargetURL = &targetURL

	if !model.StatusCode.IsNull() {
		v := client.RedirectStatusCode(int(model.StatusCode.ValueInt64()))
		out.StatusCode = &v
	}

	return out
}

func (r *BulkRedirectEntryResource) updateModelFromEntry(model *BulkRedirectEntryResourceModel, entry *client.BulkRedirectEntry) {
	model.EntryID = types.StringValue(entry.ID.String())

	if entry.Enabled != nil {
		model.Enabled = types.BoolValue(*entry.Enabled)
	} else {
		model.Enabled = types.BoolNull()
	}
	if entry.PreserveQueryString != nil {
		model.PreserveQueryString = types.BoolValue(*entry.PreserveQueryString)
	} else {
		model.PreserveQueryString = types.BoolNull()
	}
	if entry.SourceDomain != nil {
		model.SourceDomain = types.StringValue(*entry.SourceDomain)
	} else {
		model.SourceDomain = types.StringNull()
	}
	if entry.SourceScheme != nil {
		model.SourceScheme = types.StringValue(*entry.SourceScheme)
	} else {
		model.SourceScheme = types.StringNull()
	}
	if entry.SourcePath != nil {
		model.SourcePath = types.StringValue(*entry.SourcePath)
	}
	if entry.TargetURL != nil {
		model.TargetURL = types.StringValue(*entry.TargetURL)
	}
	if entry.StatusCode != nil {
		model.StatusCode = types.Int64Value(int64(*entry.StatusCode))
	} else {
		model.StatusCode = types.Int64Null()
	}
}
