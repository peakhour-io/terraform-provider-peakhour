package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

var _ resource.Resource = &RPWAFCustomRuleOrderResource{}
var _ resource.ResourceWithConfigure = &RPWAFCustomRuleOrderResource{}
var _ resource.ResourceWithImportState = &RPWAFCustomRuleOrderResource{}

func NewRPWAFCustomRuleOrderResource() resource.Resource {
	return &RPWAFCustomRuleOrderResource{}
}

type RPWAFCustomRuleOrderResource struct {
	client *client.Client
}

type RPWAFCustomRuleOrderResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Domain          types.String `tfsdk:"domain"`
	IncludeAllRules types.Bool   `tfsdk:"include_all_rules"`
	Order           types.List   `tfsdk:"order"`
}

func (r *RPWAFCustomRuleOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rp_waf_custom_rule_order"
}

func (r *RPWAFCustomRuleOrderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the order of WAF custom rules for a domain. By default it expects the complete order (include_all_rules=true), which surfaces out-of-band additions and removals as drift.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "WAF custom rule order identifier (domain/customrule_order).",
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
			"include_all_rules": schema.BoolAttribute{
				Description: "If true, order must include all WAF custom rules currently configured for the domain. If false, only the relative order of the listed rules is managed, allowing rule additions and removals in the same apply.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"order": schema.ListAttribute{
				Description: "WAF custom rule UUIDs in desired order. When include_all_rules=true, this must be the complete order. When false, it is treated as a managed subset.",
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

func (r *RPWAFCustomRuleOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RPWAFCustomRuleOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RPWAFCustomRuleOrderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating WAF custom rule order", err.Error())
		return
	}
	if err := r.readOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF custom rule order after create", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/customrule_order", plan.Domain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFCustomRuleOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RPWAFCustomRuleOrderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readOrder(ctx, &state, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF custom rule order", err.Error())
		return
	}
	state.ID = types.StringValue(fmt.Sprintf("%s/customrule_order", state.Domain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RPWAFCustomRuleOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RPWAFCustomRuleOrderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating WAF custom rule order", err.Error())
		return
	}
	if err := r.readOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading WAF custom rule order after update", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/customrule_order", plan.Domain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RPWAFCustomRuleOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op: deleting this aggregate resource does not revert ordering.
}

func (r *RPWAFCustomRuleOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected a domain name")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID+"/customrule_order")...)
}

func (r *RPWAFCustomRuleOrderResource) applyOrder(ctx context.Context, model *RPWAFCustomRuleOrderResourceModel, diags *diag.Diagnostics) error {
	if model.Order.IsUnknown() {
		return fmt.Errorf("order is unknown; ensure it references concrete custom rule UUIDs")
	}

	var desired []string
	diags.Append(model.Order.ElementsAs(ctx, &desired, false)...)
	if diags.HasError() {
		return fmt.Errorf("invalid order list")
	}

	seen := make(map[string]struct{}, len(desired))
	for _, uuid := range desired {
		if uuid == "" {
			return fmt.Errorf("order entries must be non-empty")
		}
		if _, exists := seen[uuid]; exists {
			return fmt.Errorf("order contains duplicate uuid %q", uuid)
		}
		seen[uuid] = struct{}{}
	}

	current, err := r.currentOrder(ctx, model.Domain.ValueString())
	if err != nil {
		return err
	}

	includeAll := true
	if !model.IncludeAllRules.IsNull() && !model.IncludeAllRules.IsUnknown() {
		includeAll = model.IncludeAllRules.ValueBool()
	}

	var fullOrder []string
	if includeAll {
		if len(desired) != len(current) {
			return fmt.Errorf("order must include all %d WAF custom rules (got %d)", len(current), len(desired))
		}

		currentSet := make(map[string]struct{}, len(current))
		for _, uuid := range current {
			currentSet[uuid] = struct{}{}
		}
		for _, uuid := range desired {
			if _, exists := currentSet[uuid]; !exists {
				return fmt.Errorf("order contains uuid %q which is not a WAF custom rule for this domain", uuid)
			}
		}
		fullOrder = desired
	} else {
		indexByUUID := make(map[string]int, len(current))
		for i, uuid := range current {
			indexByUUID[uuid] = i
		}

		indices := make([]int, 0, len(desired))
		for _, uuid := range desired {
			idx, exists := indexByUUID[uuid]
			if !exists {
				return fmt.Errorf("order contains uuid %q which is not a WAF custom rule for this domain", uuid)
			}
			indices = append(indices, idx)
		}
		sort.Ints(indices)

		fullOrder = append([]string(nil), current...)
		for i, idx := range indices {
			fullOrder[idx] = desired[i]
		}
	}

	return r.client.ReorderRPWAFCustomRules(model.Domain.ValueString(), fullOrder)
}

func (r *RPWAFCustomRuleOrderResource) readOrder(ctx context.Context, model *RPWAFCustomRuleOrderResourceModel, diags *diag.Diagnostics) error {
	current, err := r.currentOrder(ctx, model.Domain.ValueString())
	if err != nil {
		return err
	}

	includeAll := true
	if !model.IncludeAllRules.IsNull() && !model.IncludeAllRules.IsUnknown() {
		includeAll = model.IncludeAllRules.ValueBool()
	}

	var desiredSubset []string
	if !model.Order.IsNull() && !model.Order.IsUnknown() {
		diags.Append(model.Order.ElementsAs(ctx, &desiredSubset, false)...)
		if diags.HasError() {
			return fmt.Errorf("invalid order list")
		}
	}

	var out []string
	switch {
	case model.Order.IsNull() || model.Order.IsUnknown():
		// On initial import, populate the complete current order.
		out = current
	case includeAll:
		out = current
	default:
		managed := make(map[string]struct{}, len(desiredSubset))
		for _, uuid := range desiredSubset {
			managed[uuid] = struct{}{}
		}
		out = make([]string, 0, len(desiredSubset))
		for _, uuid := range current {
			if _, exists := managed[uuid]; exists {
				out = append(out, uuid)
			}
		}
	}

	values := make([]attr.Value, len(out))
	for i, uuid := range out {
		values[i] = types.StringValue(uuid)
	}
	model.Order = types.ListValueMust(types.StringType, values)
	return nil
}

func (r *RPWAFCustomRuleOrderResource) currentOrder(ctx context.Context, domain string) ([]string, error) {
	rules, err := r.client.ListRPWAFCustomRules(domain)
	if err != nil {
		return nil, err
	}

	sorted := append([]client.WAFCustomRule(nil), rules...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].RuleID < sorted[j].RuleID
	})

	order := make([]string, 0, len(sorted))
	for _, rule := range sorted {
		if rule.UUID != "" {
			order = append(order, rule.UUID)
		}
	}
	return order, nil
}
