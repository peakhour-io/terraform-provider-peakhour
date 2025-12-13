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

var _ resource.Resource = &RulePhaseOrderResource{}
var _ resource.ResourceWithConfigure = &RulePhaseOrderResource{}
var _ resource.ResourceWithImportState = &RulePhaseOrderResource{}

func NewRulePhaseOrderResource() resource.Resource {
	return &RulePhaseOrderResource{}
}

type RulePhaseOrderResource struct {
	client *client.Client
}

type RulePhaseOrderResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Domain          types.String `tfsdk:"domain"`
	Phase           types.String `tfsdk:"phase"`
	IncludeAllRules types.Bool   `tfsdk:"include_all_rules"`
	Order           types.List   `tfsdk:"order"`
}

func (r *RulePhaseOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_phase_order"
}

func (r *RulePhaseOrderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the order of rules within a phase for a domain using the reorder endpoint. By default it expects the full rule order for the phase (include_all_rules=true), which will surface out-of-band rule additions/removals as drift.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Rule phase order identifier (domain/phase).",
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
			"phase": schema.StringAttribute{
				Description: "Phase name (spec PhaseName enum).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"include_all_rules": schema.BoolAttribute{
				Description: "If true, order must include all rules currently in the phase (surfaces out-of-band rule changes). If false, only the relative order of the listed rules is managed.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"order": schema.ListAttribute{
				Description: "Rule UUIDs in desired order. When include_all_rules=true, this should be the full order for the phase. When false, it is treated as a subset to order relative to each other.",
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

func (r *RulePhaseOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RulePhaseOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RulePhaseOrderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error creating rule phase order", err.Error())
		return
	}

	if err := r.readOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading rule phase order after create", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.Domain.ValueString(), plan.Phase.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RulePhaseOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RulePhaseOrderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readOrder(ctx, &state, &resp.Diagnostics); err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading rule phase order", err.Error())
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/%s", state.Domain.ValueString(), state.Phase.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RulePhaseOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RulePhaseOrderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error updating rule phase order", err.Error())
		return
	}

	if err := r.readOrder(ctx, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error reading rule phase order after update", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.Domain.ValueString(), plan.Phase.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RulePhaseOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op: deleting this resource does not revert ordering.
}

func (r *RulePhaseOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'domain/phase', got %q: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("phase"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *RulePhaseOrderResource) applyOrder(ctx context.Context, model *RulePhaseOrderResourceModel, diags *diag.Diagnostics) error {
	if model.Order.IsUnknown() {
		return fmt.Errorf("order is unknown; ensure it references concrete rule UUIDs")
	}

	var desired []string
	diags.Append(model.Order.ElementsAs(ctx, &desired, false)...)
	if diags.HasError() {
		return fmt.Errorf("invalid order list")
	}

	seen := map[string]struct{}{}
	for _, uuid := range desired {
		if uuid == "" {
			return fmt.Errorf("order entries must be non-empty")
		}
		if _, ok := seen[uuid]; ok {
			return fmt.Errorf("order contains duplicate uuid %q", uuid)
		}
		seen[uuid] = struct{}{}
	}

	current, err := r.currentOrder(ctx, model.Domain.ValueString(), model.Phase.ValueString())
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
			return fmt.Errorf("order must include all %d rules currently in phase %q (got %d)", len(current), model.Phase.ValueString(), len(desired))
		}

		currentSet := map[string]struct{}{}
		for _, uuid := range current {
			currentSet[uuid] = struct{}{}
		}

		for _, uuid := range desired {
			if _, ok := currentSet[uuid]; !ok {
				return fmt.Errorf("order contains uuid %q which is not present in phase %q", uuid, model.Phase.ValueString())
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
			idx, ok := indexByUUID[uuid]
			if !ok {
				return fmt.Errorf("order contains uuid %q which is not present in phase %q", uuid, model.Phase.ValueString())
			}
			indices = append(indices, idx)
		}
		sort.Ints(indices)

		fullOrder = append([]string(nil), current...)
		for i, idx := range indices {
			fullOrder[idx] = desired[i]
		}
	}

	return r.client.ReorderRulesInPhase(model.Domain.ValueString(), model.Phase.ValueString(), fullOrder)
}

func (r *RulePhaseOrderResource) readOrder(ctx context.Context, model *RulePhaseOrderResourceModel, diags *diag.Diagnostics) error {
	current, err := r.currentOrder(ctx, model.Domain.ValueString(), model.Phase.ValueString())
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
		// On initial import, populate with the full current order so Terraform can generate config.
		out = current
	case includeAll:
		out = current
	default:
		set := map[string]struct{}{}
		for _, uuid := range desiredSubset {
			set[uuid] = struct{}{}
		}
		out = make([]string, 0, len(desiredSubset))
		for _, uuid := range current {
			if _, ok := set[uuid]; ok {
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

func (r *RulePhaseOrderResource) currentOrder(ctx context.Context, domain, phase string) ([]string, error) {
	rules, err := r.client.ListRulesInPhase(domain, phase)
	if err != nil {
		return nil, err
	}

	sorted := append([]client.RulePhaseSummary(nil), rules...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Pos < sorted[j].Pos
	})

	order := make([]string, 0, len(sorted))
	for _, rule := range sorted {
		if rule.UUID == "" {
			continue
		}
		order = append(order, rule.UUID)
	}
	return order, nil
}
