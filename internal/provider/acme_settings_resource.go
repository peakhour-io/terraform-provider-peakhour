package provider

import (
	"context"
	"fmt"

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

var _ resource.Resource = &AcmeSettingsResource{}
var _ resource.ResourceWithConfigure = &AcmeSettingsResource{}
var _ resource.ResourceWithImportState = &AcmeSettingsResource{}

func NewAcmeSettingsResource() resource.Resource {
	return &AcmeSettingsResource{}
}

type AcmeSettingsResource struct {
	client *client.Client
}

type AcmeSettingsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	DomainNames types.List   `tfsdk:"domain_names"`
}

func (r *AcmeSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acme_settings"
}

func (r *AcmeSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages ACME settings for a domain (e.g. the list of hostnames to include on an ACME certificate).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ACME settings identifier (domain/acme_settings).",
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
			"domain_names": schema.ListAttribute{
				Description: "Domain names to request via ACME. Set to null to clear.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AcmeSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AcmeSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AcmeSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating ACME settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/acme_settings")

	if err := r.readSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading ACME settings after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AcmeSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AcmeSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readSettings(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ACME settings", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/acme_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AcmeSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AcmeSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating ACME settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/acme_settings")

	if err := r.readSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading ACME settings after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AcmeSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AcmeSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateAcmeSettings(state.Domain.ValueString(), client.AcmeSettings{DomainNames: nil}); err != nil {
		resp.Diagnostics.AddError("Error deleting ACME settings", err.Error())
		return
	}
}

func (r *AcmeSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *AcmeSettingsResource) applySettings(ctx context.Context, model *AcmeSettingsResourceModel, diags *diag.Diagnostics) error {
	if model.DomainNames.IsUnknown() {
		return nil
	}

	var names []string
	if model.DomainNames.IsNull() {
		names = nil
	} else {
		diags.Append(model.DomainNames.ElementsAs(ctx, &names, false)...)
		if diags.HasError() {
			return fmt.Errorf("invalid domain_names list")
		}
	}

	return r.client.UpdateAcmeSettings(model.Domain.ValueString(), client.AcmeSettings{DomainNames: names})
}

func (r *AcmeSettingsResource) readSettings(ctx context.Context, state *AcmeSettingsResourceModel) error {
	s, err := r.client.GetAcmeSettings(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if s.DomainNames == nil {
		state.DomainNames = types.ListNull(types.StringType)
	} else {
		values := make([]attr.Value, len(s.DomainNames))
		for i, v := range s.DomainNames {
			values[i] = types.StringValue(v)
		}
		state.DomainNames = types.ListValueMust(types.StringType, values)
	}

	return nil
}
