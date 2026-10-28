package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestEnvironmentIsEdge_RequiresReplace verifies that changing is_edge replaces
// the environment (the API cannot update it), including a value that is only
// known at apply time, while an unchanged or removed value keeps the state.
func TestEnvironmentIsEdge_RequiresReplace(t *testing.T) {
	schemaResp := &resource.SchemaResponse{}
	(&EnvironmentResource{}).Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	attr := schemaResp.Schema.Attributes["is_edge"].(resourceschema.BoolAttribute)

	// Non-null resource state and plan: an update, not a create or destroy.
	raw := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})

	cases := []struct {
		name   string
		state  types.Bool
		config types.Bool
		want   bool
	}{
		{"changed", types.BoolValue(false), types.BoolValue(true), true},
		{"unchanged", types.BoolValue(true), types.BoolValue(true), false},
		{"removed from config", types.BoolValue(true), types.BoolNull(), false},
		{"unknown config", types.BoolValue(true), types.BoolUnknown(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A computed attribute absent from the config is planned as unknown.
			plan := tc.config
			if plan.IsNull() {
				plan = types.BoolUnknown()
			}
			req := planmodifier.BoolRequest{
				State:       tfsdk.State{Raw: raw},
				Plan:        tfsdk.Plan{Raw: raw},
				StateValue:  tc.state,
				PlanValue:   plan,
				ConfigValue: tc.config,
			}
			resp := &planmodifier.BoolResponse{PlanValue: plan}
			// Like the framework, each modifier sees the previous one's plan.
			for _, m := range attr.PlanModifiers {
				m.PlanModifyBool(context.Background(), req, resp)
				req.PlanValue = resp.PlanValue
			}
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}
