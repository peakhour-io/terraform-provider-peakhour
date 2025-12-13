package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

var _ resource.Resource = &RPBotsResource{}
var _ resource.ResourceWithConfigure = &RPBotsResource{}
var _ resource.ResourceWithImportState = &RPBotsResource{}

func NewRPBotsResource() resource.Resource {
	return &RPBotsResource{}
}

type RPBotsResource struct {
	client *client.Client
}

type RPBotsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Domain           types.String `tfsdk:"domain"`
	BotsInjectJS     types.Bool   `tfsdk:"bots_inject_js"`
	BotsVerifyList   types.List   `tfsdk:"bots_verify_list"`
	BotsVerifyRDNS   types.Bool   `tfsdk:"bots_verify_rdns"`
	BotsVerifyInvert types.Bool   `tfsdk:"bots_verify_invert"`
}

func (r *RPBotsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_bots"
}

func (r *RPBotsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages RP bots configuration for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP bots config identifier (domain/rp_bots).",
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
			"bots_inject_js": schema.BoolAttribute{
				Description: "Inject JS for bot detection. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"bots_verify_list": schema.ListAttribute{
				Description: "List of known bots to verify (spec enum). Set to null to clear.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"bots_verify_rdns": schema.BoolAttribute{
				Description: "Verify bots via reverse DNS. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"bots_verify_invert": schema.BoolAttribute{
				Description: "Invert bot verification logic. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPBotsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPBotsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPBotsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyBotsConfig(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP bots config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_bots")

	if err := r.readBotsConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP bots config after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPBotsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPBotsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readBotsConfig(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP bots config", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_bots")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPBotsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPBotsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyBotsConfig(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP bots config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_bots")

	if err := r.readBotsConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP bots config after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPBotsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPBotsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"bots_inject_js":     nil,
		"bots_verify_list":   nil,
		"bots_verify_rdns":   nil,
		"bots_verify_invert": nil,
	}

	if err := r.client.UpdateRPBotsConfig(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting RP bots config", err.Error())
		return
	}
}

func (r *RPBotsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPBotsResource) applyBotsConfig(ctx context.Context, model *RPBotsResourceModel, diags *diag.Diagnostics) error {
	update := map[string]any{}

	if !model.BotsInjectJS.IsUnknown() {
		if model.BotsInjectJS.IsNull() {
			update["bots_inject_js"] = nil
		} else {
			update["bots_inject_js"] = model.BotsInjectJS.ValueBool()
		}
	}

	if !model.BotsVerifyList.IsUnknown() {
		if model.BotsVerifyList.IsNull() {
			update["bots_verify_list"] = nil
		} else {
			var bots []string
			diags.Append(model.BotsVerifyList.ElementsAs(ctx, &bots, false)...)
			if diags.HasError() {
				return fmt.Errorf("invalid bots_verify_list list")
			}
			update["bots_verify_list"] = bots
		}
	}

	if !model.BotsVerifyRDNS.IsUnknown() {
		if model.BotsVerifyRDNS.IsNull() {
			update["bots_verify_rdns"] = nil
		} else {
			update["bots_verify_rdns"] = model.BotsVerifyRDNS.ValueBool()
		}
	}

	if !model.BotsVerifyInvert.IsUnknown() {
		if model.BotsVerifyInvert.IsNull() {
			update["bots_verify_invert"] = nil
		} else {
			update["bots_verify_invert"] = model.BotsVerifyInvert.ValueBool()
		}
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPBotsConfig(model.Domain.ValueString(), update)
}

func (r *RPBotsResource) readBotsConfig(ctx context.Context, state *RPBotsResourceModel) error {
	cfg, err := r.client.GetRPBotsConfig(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if cfg.BotsInjectJS != nil {
		state.BotsInjectJS = types.BoolValue(*cfg.BotsInjectJS)
	} else {
		state.BotsInjectJS = types.BoolNull()
	}

	if cfg.BotsVerifyList == nil {
		state.BotsVerifyList = types.ListNull(types.StringType)
	} else {
		values := make([]attr.Value, len(cfg.BotsVerifyList))
		for i, v := range cfg.BotsVerifyList {
			values[i] = types.StringValue(v)
		}
		state.BotsVerifyList = types.ListValueMust(types.StringType, values)
	}

	if cfg.BotsVerifyRDNS != nil {
		state.BotsVerifyRDNS = types.BoolValue(*cfg.BotsVerifyRDNS)
	} else {
		state.BotsVerifyRDNS = types.BoolNull()
	}

	if cfg.BotsVerifyInvert != nil {
		state.BotsVerifyInvert = types.BoolValue(*cfg.BotsVerifyInvert)
	} else {
		state.BotsVerifyInvert = types.BoolNull()
	}

	return nil
}
