package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const defaultsUUID = "11111111-1111-1111-1111-111111111111"
const defaultsAddress = "peakhour_rp_waf_custom_rule.test"

// These explicit server defaults are independent of the provider's normalizer.
// The API contract verifier checks them against the backing Pydantic models.
func expandWAFTestDefaults(body map[string]any) {
	fill := func(object map[string]any, defaults map[string]any) {
		for key, value := range defaults {
			if _, exists := object[key]; !exists {
				object[key] = value
			}
		}
	}
	for _, expression := range body["rules"].([]any) {
		fill(expression.(map[string]any), map[string]any{
			"variable_part": "", "variable_negated": false, "variable_counter": false,
			"variable_quote_type": nil, "operator_arg": "", "operator_negated": false, "transforms": []any{},
		})
	}
	fill(body["action"].(map[string]any), map[string]any{
		"action_arg_val": nil, "action_arg_val_param": nil, "action_arg_val_param_val": "",
	})
	fill(body["logging"].(map[string]any), map[string]any{"message": nil, "severity": nil, "tags": nil})
}

type defaultsTestAPI struct {
	mu                        sync.Mutex
	rule                      map[string]any
	creates, patches, deletes int
	renumberOnPatch           bool
}

func newDefaultsTestAPI(t *testing.T) *defaultsTestAPI {
	t.Helper()
	a := &defaultsTestAPI{rule: map[string]any{
		"uuid": defaultsUUID, "rule_id": 9, "created": "2026-07-22T00:00:00Z",
		"name": "Terraform custom rule", "description": "Terraform managed rule", "enabled": true,
		"rules": []any{
			map[string]any{"variable": "ARGS", "operator": "@contains", "operator_arg": "bad"},
			map[string]any{"variable": "REQUEST_HEADERS", "variable_part": "Host", "operator": "@contains", "operator_arg": "example"},
		},
		"action": map[string]any{"action_name": "deny"}, "logging": map[string]any{},
	}}
	expandWAFTestDefaults(a.rule)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		const endpoint = "/api/v1/domains/example.com/services/rp/waf/customrule"
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == endpoint:
			_ = json.NewEncoder(w).Encode([]any{a.rule})
		case r.Method == http.MethodPut && r.URL.Path == endpoint,
			r.Method == http.MethodPatch && r.URL.Path == endpoint+"/"+defaultsUUID:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			expandWAFTestDefaults(body)
			for _, key := range []string{"name", "description", "rules", "action", "logging"} {
				a.rule[key] = body[key]
			}
			if r.Method == http.MethodPut {
				a.creates++
				_ = json.NewEncoder(w).Encode(a.rule)
			} else {
				a.patches++
				if a.renumberOnPatch {
					a.rule["rule_id"] = 7
				}
				w.WriteHeader(http.StatusNoContent)
			}
		case r.Method == http.MethodDelete && r.URL.Path == endpoint+"/"+defaultsUUID:
			a.deletes++
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	configureMockAcceptanceTest(t, server.URL)
	return a
}

func defaultsConfig() string {
	config := testUnitWAFCustomRuleConfig(true)
	config = strings.Replace(config, `operator_arg = "bad" }`, `operator_arg = "bad" }, { variable = "REQUEST_HEADERS", variable_part = "Host", operator = "@contains", operator_arg = "example" }`, 1)
	return strings.Replace(config, `{ message = "blocked", severity = "INFO", tags = [] }`, `{}`, 1)
}

func defaultsEmptyPlan(config string) resource.TestStep {
	return resource.TestStep{Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}}
}

func (a *defaultsTestAPI) checkWrites(creates, patches int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.creates != creates || a.patches != patches {
			return fmt.Errorf("API writes: creates=%d patches=%d; want %d, %d", a.creates, a.patches, creates, patches)
		}
		return nil
	}
}

func TestAccRPWAFCustomRule_DefaultsImport(t *testing.T) {
	a := newDefaultsTestAPI(t)
	config := defaultsConfig()
	renamed := strings.Replace(config, "Terraform custom rule", "Renamed rule", 1)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, ResourceName: defaultsAddress, ImportState: true,
				ImportStateId: "example.com/customrule/" + defaultsUUID, ImportStatePersist: true},
			defaultsEmptyPlan(config),
			{RefreshState: true, Check: a.checkWrites(0, 0)},
			defaultsEmptyPlan(config),
			{Config: renamed, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(defaultsAddress, plancheck.ResourceActionUpdate),
			}}, Check: resource.ComposeTestCheckFunc(a.checkWrites(0, 1), resource.TestCheckResourceAttr(defaultsAddress, "rule_id", "9"))},
			defaultsEmptyPlan(renamed),
		},
	})
}

func TestAccRPWAFCustomRule_DefaultsDrift(t *testing.T) {
	a := newDefaultsTestAPI(t)
	config := defaultsConfig()
	steps := []resource.TestStep{{Config: config, Check: a.checkWrites(1, 0)}, defaultsEmptyPlan(config)}
	mutations := []struct {
		name   string
		change func(map[string]any)
	}{
		{"negation", func(r map[string]any) { r["rules"].([]any)[0].(map[string]any)["operator_negated"] = true }},
		{"transforms", func(r map[string]any) { r["rules"].([]any)[0].(map[string]any)["transforms"] = []any{"lowercase"} }},
		{"action argument", func(r map[string]any) { r["action"].(map[string]any)["action_arg_val_param_val"] = "password" }},
		{"explicit action null", func(r map[string]any) { r["action"].(map[string]any)["action_arg_val_param_val"] = nil }},
		{"logging", func(r map[string]any) { r["logging"].(map[string]any)["tags"] = []any{} }},
		{"unknown property", func(r map[string]any) { r["action"].(map[string]any)["future_option"] = true }},
		{"expression order", func(r map[string]any) { rules := r["rules"].([]any); rules[0], rules[1] = rules[1], rules[0] }},
	}
	for i, mutation := range mutations {
		steps = append(steps, resource.TestStep{Config: config,
			PreConfig:        func() { t.Log(mutation.name); a.mu.Lock(); defer a.mu.Unlock(); mutation.change(a.rule) },
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(defaultsAddress, plancheck.ResourceActionUpdate)}},
			Check:            a.checkWrites(1, i+1),
		}, defaultsEmptyPlan(config))
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
}

func TestAccRPWAFCustomRule_DefaultsConfigurationChanges(t *testing.T) {
	a := newDefaultsTestAPI(t)
	compact := defaultsConfig()
	explicit := strings.Replace(compact, `action_name = "deny"`, `action_name = "deny", action_arg_val = null, action_arg_val_param = null, action_arg_val_param_val = ""`, 1)
	explicit = strings.Replace(explicit, `operator_arg = "bad"`, `operator_arg = "bad", variable_part = "", variable_negated = false, variable_counter = false, operator_negated = false, variable_quote_type = null, transforms = []`, 1)
	explicit = strings.Replace(explicit, `logging_json = jsonencode({})`, `logging_json = jsonencode({ message = null, severity = null, tags = null })`, 1)
	changed := strings.Replace(compact, `operator_arg = "bad"`, `operator_arg = "bad", operator_negated = true, transforms = ["lowercase", "trim"]`, 1)
	explicitNull := strings.Replace(compact, `action_name = "deny"`, `action_name = "deny", action_arg_val_param_val = null`, 1)
	update := func(config string, patches int) resource.TestStep {
		return resource.TestStep{Config: config, Check: a.checkWrites(1, patches), ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(defaultsAddress, plancheck.ResourceActionUpdate)},
		}}
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: explicit, Check: a.checkWrites(1, 0)},
		defaultsEmptyPlan(compact), // Existing expanded state, compact config.
		defaultsEmptyPlan(explicit),
		update(changed, 1),
		defaultsEmptyPlan(changed),
		update(compact, 2), // Removing non-defaults must reset them.
		defaultsEmptyPlan(compact),
		defaultsEmptyPlan(explicit), // Existing compact state, expanded config.
		update(explicitNull, 3),     // Null differs from the declared empty-string default.
		defaultsEmptyPlan(explicitNull),
		update(compact, 4),
		defaultsEmptyPlan(compact),
	}})
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.deletes != 1 {
		t.Fatalf("destroy calls = %d, want 1", a.deletes)
	}
}

func TestAccRPWAFCustomRule_DefaultsUnknownConfiguration(t *testing.T) {
	a := newDefaultsTestAPI(t)
	config := strings.Replace(defaultsConfig(), `action_name = "deny"`, `action_name = terraform_data.action.output`, 1)
	config += `
resource "terraform_data" "action" {
  input = "deny"
}

`
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, Check: a.checkWrites(1, 0), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectUnknownValue(defaultsAddress, tfjsonpath.New("action_json")),
		}}},
		defaultsEmptyPlan(config),
		{Config: strings.Replace(config, `input = "deny"`, `input = "pass"`, 1), Check: a.checkWrites(1, 1), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectUnknownValue(defaultsAddress, tfjsonpath.New("action_json")),
			plancheck.ExpectResourceAction(defaultsAddress, plancheck.ResourceActionUpdate),
		}}},
		defaultsEmptyPlan(strings.Replace(config, `input = "deny"`, `input = "pass"`, 1)),
	}})
}

func TestAccRPWAFCustomRule_DefaultsConcurrentReorder(t *testing.T) {
	a := newDefaultsTestAPI(t)
	config := defaultsConfig()
	renamed := strings.Replace(config, "Terraform custom rule", "Renamed rule", 1)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config},
		{Config: renamed, PreConfig: func() { a.mu.Lock(); defer a.mu.Unlock(); a.renumberOnPatch = true },
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectUnknownValue(defaultsAddress, tfjsonpath.New("rule_id")),
			}}, Check: resource.ComposeTestCheckFunc(a.checkWrites(1, 1), resource.TestCheckResourceAttr(defaultsAddress, "rule_id", "7")),
		},
		defaultsEmptyPlan(renamed),
	}})
}

func TestAccRPWAFCustomRule_DefaultsImportBlock(t *testing.T) {
	a := newDefaultsTestAPI(t)
	config := defaultsConfig()
	withImport := config + fmt.Sprintf("\nimport {\n to = %s\n id = %q\n}\n", defaultsAddress, "example.com/customrule/"+defaultsUUID)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: withImport, Check: a.checkWrites(0, 0), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(defaultsAddress, plancheck.ResourceActionNoop),
		}}},
		defaultsEmptyPlan(config),
	}})
}
