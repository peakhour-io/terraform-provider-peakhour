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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

var _ resource.Resource = &RPOriginConfigResource{}
var _ resource.ResourceWithConfigure = &RPOriginConfigResource{}
var _ resource.ResourceWithImportState = &RPOriginConfigResource{}

func NewRPOriginConfigResource() resource.Resource {
	return &RPOriginConfigResource{}
}

type RPOriginConfigResource struct {
	client *client.Client
}

type RPOriginConfigResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Domain                      types.String `tfsdk:"domain"`
	SSLMode                     types.String `tfsdk:"ssl_mode"`
	OriginDowntime              types.Int64  `tfsdk:"origin_downtime"`
	OriginErrorsConnRefused     types.Int64  `tfsdk:"origin_errors_conn_refused"`
	OriginErrorsConnReset       types.Int64  `tfsdk:"origin_errors_conn_reset"`
	OriginErrorsConnTimeout     types.Int64  `tfsdk:"origin_errors_conn_timeout"`
	OriginErrorsServerError     types.Int64  `tfsdk:"origin_errors_server_error"`
	OriginErrorsResponseTimeout types.Int64  `tfsdk:"origin_errors_response_timeout"`

	OriginRequestHeaders types.Object `tfsdk:"origin_request_headers"`
}

type originRequestHeadersModel struct {
	Blocklists  types.Bool `tfsdk:"blocklists"`
	ClientProxy types.Bool `tfsdk:"client_proxy"`
	Geoip       types.Bool `tfsdk:"geoip"`
}

var originRequestHeadersTypes = map[string]attr.Type{
	"blocklists":   types.BoolType,
	"client_proxy": types.BoolType,
	"geoip":        types.BoolType,
}

func (r *RPOriginConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_origin_config"
}

func (r *RPOriginConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	useStateStr := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	useStateInt := []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	useStateBool := []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		Description: "Manages RP origin behavior settings for a domain (distinct from origin pools).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "RP origin config identifier (domain/rp_origin_config).",
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
			"ssl_mode": schema.StringAttribute{
				Description:   "Origin SSL mode (spec enum: none, https, https-client, passthrough). Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateStr,
			},
			"origin_downtime": schema.Int64Attribute{
				Description:   "Origin downtime threshold. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"origin_errors_conn_refused": schema.Int64Attribute{
				Description:   "Connection refused error threshold. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"origin_errors_conn_reset": schema.Int64Attribute{
				Description:   "Connection reset error threshold. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"origin_errors_conn_timeout": schema.Int64Attribute{
				Description:   "Connection timeout error threshold. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"origin_errors_server_error": schema.Int64Attribute{
				Description:   "5xx error threshold. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"origin_errors_response_timeout": schema.Int64Attribute{
				Description:   "Response timeout error threshold. Set to null to clear.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: useStateInt,
			},
			"origin_request_headers": schema.SingleNestedAttribute{
				Description: "Origin request header options. Set to null to clear.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"blocklists": schema.BoolAttribute{
						Description:   "Forward blocklist headers to origin. Set to null to clear.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: useStateBool,
					},
					"client_proxy": schema.BoolAttribute{
						Description:   "Forward client proxy headers to origin. Set to null to clear.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: useStateBool,
					},
					"geoip": schema.BoolAttribute{
						Description:   "Forward GeoIP headers to origin. Set to null to clear.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: useStateBool,
					},
				},
			},
		},
	}
}

func (r *RPOriginConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPOriginConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPOriginConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOriginConfig(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating RP origin config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_origin_config")

	if err := r.readOriginConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP origin config after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPOriginConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPOriginConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readOriginConfig(ctx, &state); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading RP origin config", err.Error())
		return
	}

	state.ID = types.StringValue(state.Domain.ValueString() + "/rp_origin_config")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPOriginConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPOriginConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOriginConfig(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating RP origin config", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/rp_origin_config")

	if err := r.readOriginConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading RP origin config after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPOriginConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RPOriginConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := map[string]any{
		"ssl_mode":                       nil,
		"origin_downtime":                nil,
		"origin_errors_conn_refused":     nil,
		"origin_errors_conn_reset":       nil,
		"origin_errors_conn_timeout":     nil,
		"origin_errors_server_error":     nil,
		"origin_errors_response_timeout": nil,
		"origin_request_headers":         nil,
	}

	if err := r.client.UpdateRPOriginConfig(state.Domain.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Error deleting RP origin config", err.Error())
		return
	}
}

func (r *RPOriginConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

func (r *RPOriginConfigResource) applyOriginConfig(ctx context.Context, model *RPOriginConfigResourceModel, diags *diag.Diagnostics) error {
	update := map[string]any{}

	if !model.SSLMode.IsUnknown() {
		if model.SSLMode.IsNull() {
			update["ssl_mode"] = nil
		} else {
			update["ssl_mode"] = model.SSLMode.ValueString()
		}
	}
	if !model.OriginDowntime.IsUnknown() {
		if model.OriginDowntime.IsNull() {
			update["origin_downtime"] = nil
		} else {
			update["origin_downtime"] = int(model.OriginDowntime.ValueInt64())
		}
	}
	if !model.OriginErrorsConnRefused.IsUnknown() {
		if model.OriginErrorsConnRefused.IsNull() {
			update["origin_errors_conn_refused"] = nil
		} else {
			update["origin_errors_conn_refused"] = int(model.OriginErrorsConnRefused.ValueInt64())
		}
	}
	if !model.OriginErrorsConnReset.IsUnknown() {
		if model.OriginErrorsConnReset.IsNull() {
			update["origin_errors_conn_reset"] = nil
		} else {
			update["origin_errors_conn_reset"] = int(model.OriginErrorsConnReset.ValueInt64())
		}
	}
	if !model.OriginErrorsConnTimeout.IsUnknown() {
		if model.OriginErrorsConnTimeout.IsNull() {
			update["origin_errors_conn_timeout"] = nil
		} else {
			update["origin_errors_conn_timeout"] = int(model.OriginErrorsConnTimeout.ValueInt64())
		}
	}
	if !model.OriginErrorsServerError.IsUnknown() {
		if model.OriginErrorsServerError.IsNull() {
			update["origin_errors_server_error"] = nil
		} else {
			update["origin_errors_server_error"] = int(model.OriginErrorsServerError.ValueInt64())
		}
	}
	if !model.OriginErrorsResponseTimeout.IsUnknown() {
		if model.OriginErrorsResponseTimeout.IsNull() {
			update["origin_errors_response_timeout"] = nil
		} else {
			update["origin_errors_response_timeout"] = int(model.OriginErrorsResponseTimeout.ValueInt64())
		}
	}

	if !model.OriginRequestHeaders.IsUnknown() {
		if model.OriginRequestHeaders.IsNull() {
			update["origin_request_headers"] = nil
		} else {
			var hdr originRequestHeadersModel
			diags.Append(model.OriginRequestHeaders.As(ctx, &hdr, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return fmt.Errorf("invalid origin_request_headers")
			}

			nested := map[string]any{}
			if !hdr.Blocklists.IsUnknown() {
				if hdr.Blocklists.IsNull() {
					nested["blocklists"] = nil
				} else {
					nested["blocklists"] = hdr.Blocklists.ValueBool()
				}
			}
			if !hdr.ClientProxy.IsUnknown() {
				if hdr.ClientProxy.IsNull() {
					nested["client_proxy"] = nil
				} else {
					nested["client_proxy"] = hdr.ClientProxy.ValueBool()
				}
			}
			if !hdr.Geoip.IsUnknown() {
				if hdr.Geoip.IsNull() {
					nested["geoip"] = nil
				} else {
					nested["geoip"] = hdr.Geoip.ValueBool()
				}
			}

			if len(nested) == 0 {
				// nothing to change
			} else {
				update["origin_request_headers"] = nested
			}
		}
	}

	if len(update) == 0 {
		return nil
	}

	return r.client.UpdateRPOriginConfig(model.Domain.ValueString(), update)
}

func (r *RPOriginConfigResource) readOriginConfig(ctx context.Context, state *RPOriginConfigResourceModel) error {
	cfg, err := r.client.GetRPOriginConfig(state.Domain.ValueString())
	if err != nil {
		return err
	}

	if cfg.SSLMode != nil {
		state.SSLMode = types.StringValue(*cfg.SSLMode)
	} else {
		state.SSLMode = types.StringNull()
	}
	if cfg.OriginDowntime != nil {
		state.OriginDowntime = types.Int64Value(int64(*cfg.OriginDowntime))
	} else {
		state.OriginDowntime = types.Int64Null()
	}
	if cfg.OriginErrorsConnRefused != nil {
		state.OriginErrorsConnRefused = types.Int64Value(int64(*cfg.OriginErrorsConnRefused))
	} else {
		state.OriginErrorsConnRefused = types.Int64Null()
	}
	if cfg.OriginErrorsConnReset != nil {
		state.OriginErrorsConnReset = types.Int64Value(int64(*cfg.OriginErrorsConnReset))
	} else {
		state.OriginErrorsConnReset = types.Int64Null()
	}
	if cfg.OriginErrorsConnTimeout != nil {
		state.OriginErrorsConnTimeout = types.Int64Value(int64(*cfg.OriginErrorsConnTimeout))
	} else {
		state.OriginErrorsConnTimeout = types.Int64Null()
	}
	if cfg.OriginErrorsServerError != nil {
		state.OriginErrorsServerError = types.Int64Value(int64(*cfg.OriginErrorsServerError))
	} else {
		state.OriginErrorsServerError = types.Int64Null()
	}
	if cfg.OriginErrorsResponseTimeout != nil {
		state.OriginErrorsResponseTimeout = types.Int64Value(int64(*cfg.OriginErrorsResponseTimeout))
	} else {
		state.OriginErrorsResponseTimeout = types.Int64Null()
	}

	if cfg.OriginRequestHeaders == nil {
		state.OriginRequestHeaders = types.ObjectNull(originRequestHeadersTypes)
	} else {
		attrs := map[string]attr.Value{
			"blocklists":   types.BoolNull(),
			"client_proxy": types.BoolNull(),
			"geoip":        types.BoolNull(),
		}
		if cfg.OriginRequestHeaders.Blocklists != nil {
			attrs["blocklists"] = types.BoolValue(*cfg.OriginRequestHeaders.Blocklists)
		}
		if cfg.OriginRequestHeaders.ClientProxy != nil {
			attrs["client_proxy"] = types.BoolValue(*cfg.OriginRequestHeaders.ClientProxy)
		}
		if cfg.OriginRequestHeaders.GeoIP != nil {
			attrs["geoip"] = types.BoolValue(*cfg.OriginRequestHeaders.GeoIP)
		}
		state.OriginRequestHeaders = types.ObjectValueMust(originRequestHeadersTypes, attrs)
	}

	return nil
}
