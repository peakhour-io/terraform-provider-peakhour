package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPLuaOptionsResource{}
var _ resource.ResourceWithConfigure = &RPLuaOptionsResource{}
var _ resource.ResourceWithImportState = &RPLuaOptionsResource{}

func NewRPLuaOptionsResource() resource.Resource {
	return &RPLuaOptionsResource{}
}

type RPLuaOptionsResource struct {
	client *client.Client
}

type RPLuaOptionsResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Domain                  types.String `tfsdk:"domain"`
	LuaEnabled              types.Bool   `tfsdk:"lua_enabled"`
	LuaRequestFilter        types.String `tfsdk:"lua_request_filter"`
	LuaResponseFilter       types.String `tfsdk:"lua_response_filter"`
	LuaOriginRequestFilter  types.String `tfsdk:"lua_origin_request_filter"`
	LuaOriginResponseFilter types.String `tfsdk:"lua_origin_response_filter"`
	LuaOriginSelector       types.String `tfsdk:"lua_origin_selector"`
	LuaOriginPoolSelector   types.String `tfsdk:"lua_origin_pool_selector"`
}

func (r *RPLuaOptionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_lua_options"
}

func (r *RPLuaOptionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	useStateBool := []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}
	useStateStr := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		Description: "Manages RP Lua options for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP Lua options identifier (domain/rp_lua_options).",
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
			"lua_enabled": schema.BoolAttribute{
				Description:   "Whether Lua processing is enabled. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateBool,
			},
			"lua_request_filter": schema.StringAttribute{
				Description:   "Lua request filter. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"lua_response_filter": schema.StringAttribute{
				Description:   "Lua response filter. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"lua_origin_request_filter": schema.StringAttribute{
				Description:   "Lua origin request filter. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"lua_origin_response_filter": schema.StringAttribute{
				Description:   "Lua origin response filter. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"lua_origin_selector": schema.StringAttribute{
				Description:   "Lua origin selector. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"lua_origin_pool_selector": schema.StringAttribute{
				Description:   "Lua origin pool selector. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
		},
	}
}

func (r *RPLuaOptionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPLuaOptionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPLuaOptionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyLuaOptions(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP Lua options", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_lua_options")

	if err := r.readLuaOptions(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP Lua options after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPLuaOptionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPLuaOptionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readLuaOptions(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP Lua options", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_lua_options")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPLuaOptionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPLuaOptionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyLuaOptions(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP Lua options", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_lua_options")

	if err := r.readLuaOptions(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP Lua options after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPLuaOptionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPLuaOptionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateRPLuaOptions(state.Domain.ValueString(), client.LuaOptions{}); err != nil {
		resp.Diagnostics.AddError("Error deleting RP Lua options", err.Error())
		return
	}
}

func (r *RPLuaOptionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func mergeBool(plan types.Bool, current *bool) *bool {
	if plan.IsUnknown() {
		return current
	}
	if plan.IsNull() {
		return nil
	}
	v := plan.ValueBool()
	return &v
}

func mergeString(plan types.String, current *string) *string {
	if plan.IsUnknown() {
		return current
	}
	if plan.IsNull() {
		return nil
	}
	v := plan.ValueString()
	return &v
}

func (r *RPLuaOptionsResource) applyLuaOptions(ctx context.Context, model *RPLuaOptionsResourceModel, diags *diag.Diagnostics) error {
	current, err := r.client.GetRPLuaOptions(model.Domain.ValueString())
	if err != nil {
		if client.IsNotFoundError(err) {
			current = &client.LuaOptions{}
		} else {
			return err
		}
	}

	merged := client.LuaOptions{
		LuaEnabled:              mergeBool(model.LuaEnabled, current.LuaEnabled),
		LuaRequestFilter:        mergeString(model.LuaRequestFilter, current.LuaRequestFilter),
		LuaResponseFilter:       mergeString(model.LuaResponseFilter, current.LuaResponseFilter),
		LuaOriginRequestFilter:  mergeString(model.LuaOriginRequestFilter, current.LuaOriginRequestFilter),
		LuaOriginResponseFilter: mergeString(model.LuaOriginResponseFilter, current.LuaOriginResponseFilter),
		LuaOriginSelector:       mergeString(model.LuaOriginSelector, current.LuaOriginSelector),
		LuaOriginPoolSelector:   mergeString(model.LuaOriginPoolSelector, current.LuaOriginPoolSelector),
	}

	return r.client.UpdateRPLuaOptions(model.Domain.ValueString(), merged)
}

func (r *RPLuaOptionsResource) readLuaOptions(ctx context.Context, state *RPLuaOptionsResourceModel) error {
	cfg, err := r.client.GetRPLuaOptions(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if cfg.LuaEnabled != nil {
		state.LuaEnabled = types.BoolValue(*cfg.LuaEnabled)
	} else {
		state.LuaEnabled = types.BoolNull()
	}
	if cfg.LuaRequestFilter != nil {
		state.LuaRequestFilter = types.StringValue(*cfg.LuaRequestFilter)
	} else {
		state.LuaRequestFilter = types.StringNull()
	}
	if cfg.LuaResponseFilter != nil {
		state.LuaResponseFilter = types.StringValue(*cfg.LuaResponseFilter)
	} else {
		state.LuaResponseFilter = types.StringNull()
	}
	if cfg.LuaOriginRequestFilter != nil {
		state.LuaOriginRequestFilter = types.StringValue(*cfg.LuaOriginRequestFilter)
	} else {
		state.LuaOriginRequestFilter = types.StringNull()
	}
	if cfg.LuaOriginResponseFilter != nil {
		state.LuaOriginResponseFilter = types.StringValue(*cfg.LuaOriginResponseFilter)
	} else {
		state.LuaOriginResponseFilter = types.StringNull()
	}
	if cfg.LuaOriginSelector != nil {
		state.LuaOriginSelector = types.StringValue(*cfg.LuaOriginSelector)
	} else {
		state.LuaOriginSelector = types.StringNull()
	}
	if cfg.LuaOriginPoolSelector != nil {
		state.LuaOriginPoolSelector = types.StringValue(*cfg.LuaOriginPoolSelector)
	} else {
		state.LuaOriginPoolSelector = types.StringNull()
	}

	return nil
}
