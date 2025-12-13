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

var (
	_ resource.Resource                = &ReverseProxyServiceResource{}
	_ resource.ResourceWithConfigure   = &ReverseProxyServiceResource{}
	_ resource.ResourceWithImportState = &ReverseProxyServiceResource{}
)

func NewReverseProxyServiceResource() resource.Resource {
	return &ReverseProxyServiceResource{}
}

type ReverseProxyServiceResource struct {
	client *client.Client
}

type ReverseProxyServiceResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`
}

func (r *ReverseProxyServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_proxy_service"
}

func (r *ReverseProxyServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Enables the Reverse Proxy (CDN) service for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "Domain name to enable the service for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ReverseProxyServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReverseProxyServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReverseProxyServiceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create service via API
	err := r.client.CreateDomainService(plan.Domain.ValueString(), "rp")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating reverse proxy service",
			"Could not create service for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp")

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ReverseProxyServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ReverseProxyServiceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if service exists
	err := r.client.GetDomainService(state.Domain.ValueString(), "rp")
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading reverse proxy service",
			"Could not read service for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp")

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ReverseProxyServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Service cannot be updated (domain requires replace)
	var plan ReverseProxyServiceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ReverseProxyServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ReverseProxyServiceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete service via API
	err := r.client.DeleteDomainService(state.Domain.ValueString(), "rp")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting reverse proxy service",
			"Could not delete service for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *ReverseProxyServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the domain name
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}
