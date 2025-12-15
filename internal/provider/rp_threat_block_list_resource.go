package provider

import (
	"context"
	"fmt"
	"sort"

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

var _ resource.Resource = &RPThreatBlockListResource{}
var _ resource.ResourceWithConfigure = &RPThreatBlockListResource{}
var _ resource.ResourceWithImportState = &RPThreatBlockListResource{}

func NewRPThreatBlockListResource() resource.Resource {
	return &RPThreatBlockListResource{}
}

type RPThreatBlockListResource struct {
	client *client.Client
}

type RPThreatBlockListResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Domain     types.String `tfsdk:"domain"`
	Blocklists types.List   `tfsdk:"blocklists"`
}

func (r *RPThreatBlockListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_threat_block_list"
}

func (r *RPThreatBlockListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages enabled reverse proxy threat block lists for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Threat block list identifier (domain/rp_threat_block_list).",
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
			"blocklists": schema.ListAttribute{
				Description: "Names of block lists to enable (the API will disable any others).",
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPThreatBlockListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPThreatBlockListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPThreatBlockListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating threat block list config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_threat_block_list")
	// Don't call read() - trust the plan values. Read() will refresh on next plan.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPThreatBlockListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPThreatBlockListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.read(ctx, &state, &resp.Diagnostics); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading threat block list config", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_threat_block_list")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPThreatBlockListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPThreatBlockListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating threat block list config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_threat_block_list")
	// Don't call read() - trust the plan values. Read() will refresh on next plan.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPThreatBlockListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPThreatBlockListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetThreatBlockLists(state.Domain.ValueString(), client.BlocklistsSet{Blocklists: []string{}}); err != nil {
		resp.Diagnostics.AddError("Error deleting threat block list config", err.Error())
		return
	}
}

func (r *RPThreatBlockListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPThreatBlockListResource) apply(ctx context.Context, model *RPThreatBlockListResourceModel, diags *diag.Diagnostics) error {
	var enabled []string
	diags.Append(model.Blocklists.ElementsAs(ctx, &enabled, false)...)
	if diags.HasError() {
		return fmt.Errorf("invalid blocklists list")
	}

	sort.Strings(enabled)

	return r.client.SetThreatBlockLists(model.Domain.ValueString(), client.BlocklistsSet{Blocklists: enabled})
}

func (r *RPThreatBlockListResource) read(ctx context.Context, model *RPThreatBlockListResourceModel, diags *diag.Diagnostics) error {
	lists, err := r.client.ListThreatBlockLists(model.Domain.ValueString())
	if err != nil {
		return err
	}

	enabled := make([]string, 0, len(lists))
	for _, l := range lists {
		if l.Enabled {
			enabled = append(enabled, l.Name)
		}
	}
	sort.Strings(enabled)

	values := make([]attr.Value, len(enabled))
	for i, name := range enabled {
		values[i] = types.StringValue(name)
	}
	model.Blocklists = types.ListValueMust(types.StringType, values)
	return nil
}
