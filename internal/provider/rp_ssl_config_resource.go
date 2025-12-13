package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPSSLConfigResource{}
var _ resource.ResourceWithConfigure = &RPSSLConfigResource{}
var _ resource.ResourceWithImportState = &RPSSLConfigResource{}

func NewRPSSLConfigResource() resource.Resource {
	return &RPSSLConfigResource{}
}

type RPSSLConfigResource struct {
	client *client.Client
}

type RPSSLConfigResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Ciphers types.String `tfsdk:"ciphers"`
}

func (r *RPSSLConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_ssl_config"
}

func (r *RPSSLConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages RP SSL/TLS settings for a domain (cipher profile).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP SSL config identifier (domain/rp_ssl_config).",
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
			"ciphers": schema.StringAttribute{
				Description: "Cipher profile (spec enum: intermediate, modern, old).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPSSLConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPSSLConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPSSLConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySSLConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating RP SSL config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_ssl_config")

	if err := r.readSSLConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP SSL config after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPSSLConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPSSLConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readSSLConfig(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP SSL config", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_ssl_config")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPSSLConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPSSLConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySSLConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating RP SSL config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_ssl_config")

	if err := r.readSSLConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP SSL config after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPSSLConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No server-side "delete"; removing from Terraform leaves the remote configuration unchanged.
}

func (r *RPSSLConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPSSLConfigResource) applySSLConfig(ctx context.Context, model *RPSSLConfigResourceModel) error {
	if model.Ciphers.IsUnknown() || model.Ciphers.IsNull() {
		return nil
	}

	ciphers := strings.TrimSpace(model.Ciphers.ValueString())
	if ciphers == "" {
		return fmt.Errorf("ciphers must not be empty")
	}

	return r.client.UpdateRPSSLConfig(model.Domain.ValueString(), client.SSLConfig{Ciphers: ciphers})
}

func (r *RPSSLConfigResource) readSSLConfig(ctx context.Context, state *RPSSLConfigResourceModel) error {
	cfg, err := r.client.GetRPSSLConfig(state.Domain.ValueString())
	if err != nil {
		return err
	}

	state.Ciphers = types.StringValue(cfg.Ciphers)
	return nil
}
