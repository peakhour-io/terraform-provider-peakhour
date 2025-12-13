package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &AcmeCertificateResource{}
var _ resource.ResourceWithConfigure = &AcmeCertificateResource{}
var _ resource.ResourceWithImportState = &AcmeCertificateResource{}

func NewAcmeCertificateResource() resource.Resource {
	return &AcmeCertificateResource{}
}

type AcmeCertificateResource struct {
	client *client.Client
}

type AcmeCertificateResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	Issue          types.Bool   `tfsdk:"issue"`
	State          types.String `tfsdk:"state"`
	NotBefore      types.String `tfsdk:"not_before"`
	NotAfter       types.String `tfsdk:"not_after"`
	CertificatePEM types.String `tfsdk:"certificate_pem"`
}

func (r *AcmeCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acme_certificate"
}

func (r *AcmeCertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads ACME certificate status for a domain and can optionally trigger issuance. The issuance request is asynchronous.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ACME certificate identifier (domain/acme_certificate).",
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
			"issue": schema.BoolAttribute{
				Description: "When set to true, triggers an ACME issuance request on create/update (async). Toggle from false→true to re-issue.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Description: "ACME certificate state (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"not_before": schema.StringAttribute{
				Description: "Certificate validity start time (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"not_after": schema.StringAttribute{
				Description: "Certificate validity end time (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate_pem": schema.StringAttribute{
				Description: "Certificate PEM (computed, may be null).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AcmeCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AcmeCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AcmeCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.maybeIssueCertificate(ctx, plan.Domain.ValueString(), plan.Issue, types.BoolNull(), &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating ACME certificate", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/acme_certificate")

	if err := r.readCertificate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading ACME certificate after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AcmeCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AcmeCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readCertificate(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ACME certificate", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/acme_certificate")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AcmeCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AcmeCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior AcmeCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.maybeIssueCertificate(ctx, plan.Domain.ValueString(), plan.Issue, prior.Issue, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating ACME certificate", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/acme_certificate")

	if err := r.readCertificate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading ACME certificate after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AcmeCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No server-side delete.
}

func (r *AcmeCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *AcmeCertificateResource) maybeIssueCertificate(ctx context.Context, domain string, issue types.Bool, prior types.Bool, diags *diag.Diagnostics) error {
	if issue.IsUnknown() || issue.IsNull() {
		return nil
	}
	if !issue.ValueBool() {
		return nil
	}

	// Only issue on false->true transition when prior is known; for Create, prior is null.
	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueBool() {
		return nil
	}

	return r.client.IssueAcmeCertificate(domain)
}

func (r *AcmeCertificateResource) readCertificate(ctx context.Context, state *AcmeCertificateResourceModel) error {
	cert, err := r.client.GetAcmeCertificate(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if cert.State != nil {
		state.State = types.StringValue(*cert.State)
	} else {
		state.State = types.StringNull()
	}
	if cert.NotBefore != nil {
		state.NotBefore = types.StringValue(*cert.NotBefore)
	} else {
		state.NotBefore = types.StringNull()
	}
	if cert.NotAfter != nil {
		state.NotAfter = types.StringValue(*cert.NotAfter)
	} else {
		state.NotAfter = types.StringNull()
	}
	if cert.CertificatePEM != nil {
		state.CertificatePEM = types.StringValue(*cert.CertificatePEM)
	} else {
		state.CertificatePEM = types.StringNull()
	}

	return nil
}
