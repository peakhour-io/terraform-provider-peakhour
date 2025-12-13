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

var _ resource.Resource = &RPSSLCertificateResource{}
var _ resource.ResourceWithConfigure = &RPSSLCertificateResource{}
var _ resource.ResourceWithImportState = &RPSSLCertificateResource{}

func NewRPSSLCertificateResource() resource.Resource {
	return &RPSSLCertificateResource{}
}

type RPSSLCertificateResource struct {
	client *client.Client
}

type RPSSLCertificateResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Domain types.String `tfsdk:"domain"`

	Verify         types.Bool   `tfsdk:"verify"`
	CertificatePEM types.String `tfsdk:"certificate_pem"`
	PrivateKeyPEM  types.String `tfsdk:"private_key_pem"`

	CN        types.String `tfsdk:"cn"`
	AltName   types.String `tfsdk:"alt_name"`
	Issuer    types.String `tfsdk:"issuer"`
	ValidFrom types.String `tfsdk:"valid_from"`
	ValidTo   types.String `tfsdk:"valid_to"`
}

func (r *RPSSLCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_ssl_certificate"
}

func (r *RPSSLCertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Uploads and tracks the currently configured RP SSL certificate for a domain. Note: the API does not return the uploaded private key (or certificate PEM), so Terraform cannot automatically verify drift of those values.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP SSL certificate identifier (domain/rp_ssl_certificate).",
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
			"verify": schema.BoolAttribute{
				Description: "Whether to verify the certificate/private key on upload (default: true).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate_pem": schema.StringAttribute{
				Description: "Certificate PEM to upload. Optional; when unset, Terraform will only read certificate info from the API.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key_pem": schema.StringAttribute{
				Description: "Private key PEM to upload. Optional; required if certificate_pem is set. Stored in Terraform state (sensitive).",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cn": schema.StringAttribute{
				Description: "Certificate common name (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"alt_name": schema.StringAttribute{
				Description: "Certificate subjectAltName (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"issuer": schema.StringAttribute{
				Description: "Certificate issuer (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"valid_from": schema.StringAttribute{
				Description: "Certificate validity start time (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"valid_to": schema.StringAttribute{
				Description: "Certificate validity end time (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPSSLCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPSSLCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPSSLCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.maybeUploadCertificate(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP SSL certificate", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_ssl_certificate")

	if err := r.readCertificateInfo(ctx, &plan); err != nil {
		noUploadConfigured := (plan.CertificatePEM.IsNull() || plan.CertificatePEM.IsUnknown()) &&
			(plan.PrivateKeyPEM.IsNull() || plan.PrivateKeyPEM.IsUnknown())
		if client.IsNotFoundError(err) && noUploadConfigured {
			resp.Diagnostics.AddError(
				"SSL certificate not found",
				"No SSL certificate exists for this domain. To create one, set certificate_pem and private_key_pem.",
			)
			return
		}
		resp.Diagnostics.AddError("Error reading RP SSL certificate after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPSSLCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPSSLCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readCertificateInfo(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP SSL certificate", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_ssl_certificate")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPSSLCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPSSLCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior RPSSLCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.maybeUploadCertificate(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP SSL certificate", err.Error())
		return
	}

	plan.CertificatePEM = mergeStringValue(plan.CertificatePEM, prior.CertificatePEM)
	plan.PrivateKeyPEM = mergeStringValue(plan.PrivateKeyPEM, prior.PrivateKeyPEM)
	plan.Verify = mergeBoolValue(plan.Verify, prior.Verify)

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_ssl_certificate")

	if err := r.readCertificateInfo(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP SSL certificate after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPSSLCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No server-side delete.
}

func (r *RPSSLCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func mergeStringValue(plan types.String, prior types.String) types.String {
	if plan.IsUnknown() {
		return prior
	}
	return plan
}

func mergeBoolValue(plan types.Bool, prior types.Bool) types.Bool {
	if plan.IsUnknown() {
		return prior
	}
	return plan
}

func (r *RPSSLCertificateResource) maybeUploadCertificate(ctx context.Context, model *RPSSLCertificateResourceModel, diags *diag.Diagnostics) error {
	certKnown := !model.CertificatePEM.IsUnknown()
	keyKnown := !model.PrivateKeyPEM.IsUnknown()

	// If neither is set in config, do not upload.
	if !certKnown && !keyKnown {
		return nil
	}

	// If both explicitly set to null, clear local state only (no API call).
	if certKnown && keyKnown && model.CertificatePEM.IsNull() && model.PrivateKeyPEM.IsNull() {
		return nil
	}

	if !certKnown || !keyKnown {
		diags.AddError(
			"Invalid SSL certificate configuration",
			"certificate_pem and private_key_pem must be set together (or both omitted).",
		)
		return fmt.Errorf("certificate_pem and private_key_pem must be set together")
	}

	if model.CertificatePEM.IsNull() || model.PrivateKeyPEM.IsNull() {
		diags.AddError(
			"Invalid SSL certificate configuration",
			"certificate_pem and private_key_pem must either both be null (to clear local state) or both be non-null (to upload).",
		)
		return fmt.Errorf("certificate_pem and private_key_pem must both be null or both be non-null")
	}

	var verify *bool
	if !model.Verify.IsNull() && !model.Verify.IsUnknown() {
		v := model.Verify.ValueBool()
		verify = &v
	}

	return r.client.UpdateRPSSLCertificate(model.Domain.ValueString(), client.SSLCertificateAdd{
		Verify:      verify,
		Certificate: model.CertificatePEM.ValueString(),
		PrivateKey:  model.PrivateKeyPEM.ValueString(),
	})
}

func (r *RPSSLCertificateResource) readCertificateInfo(ctx context.Context, state *RPSSLCertificateResourceModel) error {
	cert, err := r.client.GetRPSSLCertificate(state.Domain.ValueString())
	if err != nil {
		return err
	}

	state.CN = types.StringValue(cert.Certificate.CN)
	state.AltName = types.StringValue(cert.Certificate.AltName)
	state.Issuer = types.StringValue(cert.Certificate.Issuer)
	state.ValidFrom = types.StringValue(cert.Certificate.ValidFrom)
	state.ValidTo = types.StringValue(cert.Certificate.ValidTo)
	return nil
}
