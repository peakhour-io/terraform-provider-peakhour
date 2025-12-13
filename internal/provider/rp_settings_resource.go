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

var _ resource.Resource = &RPSettingsResource{}
var _ resource.ResourceWithConfigure = &RPSettingsResource{}
var _ resource.ResourceWithImportState = &RPSettingsResource{}

func NewRPSettingsResource() resource.Resource {
	return &RPSettingsResource{}
}

type RPSettingsResource struct {
	client *client.Client
}

type RPSettingsResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	NotificationEmails types.List   `tfsdk:"notification_emails"`
	Quickstart         types.Bool   `tfsdk:"quickstart"`
	IPv4Address        types.String `tfsdk:"ipv4_address"`
	IPv6Address        types.String `tfsdk:"ipv6_address"`
	CNAME              types.String `tfsdk:"cname"`
}

func (r *RPSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_settings"
}

func (r *RPSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages reverse proxy (RP) service settings for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP settings identifier (domain/rp_settings).",
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
			"notification_emails": schema.ListAttribute{
				Description: "Notification email addresses. Set to null to clear.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"quickstart": schema.BoolAttribute{
				Description: "Whether the domain is in quickstart mode. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv4_address": schema.StringAttribute{
				Description: "Assigned IPv4 address (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_address": schema.StringAttribute{
				Description: "Assigned IPv6 address (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cname": schema.StringAttribute{
				Description: "Assigned CNAME (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RPSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_settings")

	if err := r.readSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP settings after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readSettings(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP settings", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySettings(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP settings", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_settings")

	if err := r.readSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP settings after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"notification_emails": nil,
		"quickstart":          nil,
	}

	if err := r.client.UpdateRPSettings(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting RP settings", err.Error())
		return
	}
}

func (r *RPSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPSettingsResource) applySettings(ctx context.Context, model *RPSettingsResourceModel, diags *diag.Diagnostics) error {
	update := map[string]any{}

	if !model.NotificationEmails.IsUnknown() {
		if model.NotificationEmails.IsNull() {
			update["notification_emails"] = nil
		} else {
			var emails []string
			diags.Append(model.NotificationEmails.ElementsAs(ctx, &emails, false)...)
			if diags.HasError() {
				return fmt.Errorf("invalid notification_emails list")
			}
			update["notification_emails"] = emails
		}
	}

	if !model.Quickstart.IsUnknown() {
		if model.Quickstart.IsNull() {
			update["quickstart"] = nil
		} else {
			update["quickstart"] = model.Quickstart.ValueBool()
		}
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPSettings(model.Domain.ValueString(), update)
}

func (r *RPSettingsResource) readSettings(ctx context.Context, state *RPSettingsResourceModel) error {
	s, err := r.client.GetRPSettings(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if s.NotificationEmails == nil {
		state.NotificationEmails = types.ListNull(types.StringType)
	} else {
		values := make([]attr.Value, len(s.NotificationEmails))
		for i, v := range s.NotificationEmails {
			values[i] = types.StringValue(v)
		}
		state.NotificationEmails = types.ListValueMust(types.StringType, values)
	}

	if s.Quickstart != nil {
		state.Quickstart = types.BoolValue(*s.Quickstart)
	} else {
		state.Quickstart = types.BoolNull()
	}

	if s.IPv4Address != nil {
		state.IPv4Address = types.StringValue(*s.IPv4Address)
	} else {
		state.IPv4Address = types.StringNull()
	}

	if s.IPv6Address != nil {
		state.IPv6Address = types.StringValue(*s.IPv6Address)
	} else {
		state.IPv6Address = types.StringNull()
	}

	if s.CNAME != nil {
		state.CNAME = types.StringValue(*s.CNAME)
	} else {
		state.CNAME = types.StringNull()
	}

	return nil
}
