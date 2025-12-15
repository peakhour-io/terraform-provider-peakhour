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

var _ resource.Resource = &DomainPlanResource{}
var _ resource.ResourceWithConfigure = &DomainPlanResource{}
var _ resource.ResourceWithImportState = &DomainPlanResource{}

func NewDomainPlanResource() resource.Resource {
	return &DomainPlanResource{}
}

type DomainPlanResource struct {
	client *client.Client
}

type DomainPlanResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	Code        types.String `tfsdk:"code"`
	UseExisting types.Bool   `tfsdk:"use_existing"`
}

func (r *DomainPlanResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_plan"
}

func (r *DomainPlanResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a subscription plan to a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Domain plan identifier (domain/plan).",
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
			"code": schema.StringAttribute{
				Description: "Plan code to assign to the domain (e.g. 'basic', 'premium').",
				Required:    true,
			},
			"use_existing": schema.BoolAttribute{
				Description: "If true, subscribe to an existing account plan. If false (default), create a new plan subscription.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *DomainPlanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainPlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainPlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if plan.UseExisting.ValueBool() {
		err = r.client.SetDomainPlanExisting(plan.Domain.ValueString(), plan.Code.ValueString())
	} else {
		err = r.client.SetDomainPlanNew(plan.Domain.ValueString(), plan.Code.ValueString())
	}
	if err != nil {
		if client.IsConflictError(err) {
			resp.Diagnostics.AddError(
				"Domain Plan Already Assigned",
				fmt.Sprintf("A plan is already assigned to domain %q. To manage it with Terraform, add an import block:\n\n"+
					"  import {\n"+
					"    to = peakhour_domain_plan.%s\n"+
					"    id = %q\n"+
					"  }\n\n"+
					"Then run: terraform apply",
					plan.Domain.ValueString(), "example", plan.Domain.ValueString()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Error assigning plan to domain",
				"Could not assign plan to domain "+plan.Domain.ValueString()+": "+err.Error(),
			)
		}
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/plan")

	if err := r.readPlan(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading domain plan after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainPlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainPlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readPlan(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading domain plan",
			"Could not read plan for domain "+state.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/plan")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainPlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainPlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if plan.UseExisting.ValueBool() {
		err = r.client.SetDomainPlanExisting(plan.Domain.ValueString(), plan.Code.ValueString())
	} else {
		err = r.client.SetDomainPlanNew(plan.Domain.ValueString(), plan.Code.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating domain plan",
			"Could not update plan for domain "+plan.Domain.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/plan")

	if err := r.readPlan(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading domain plan after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainPlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainPlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UnsubscribeDomainPlan(state.Domain.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error unsubscribing domain from plan",
			"Could not unsubscribe domain "+state.Domain.ValueString()+" from plan: "+err.Error(),
		)
		return
	}
}

func (r *DomainPlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *DomainPlanResource) readPlan(ctx context.Context, model *DomainPlanResourceModel) error {
	domainPlan, err := r.client.GetDomainPlan(model.Domain.ValueString())
	if err != nil {
		return err
	}

	model.Code = types.StringValue(domainPlan.Plan.Code)
	return nil
}
