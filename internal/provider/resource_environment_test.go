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
// the environment (the API cannot update it), while a null state after import,
// a removed or unknown value, or an unchanged value does not.
func TestEnvironmentIsEdge_RequiresReplace(t *testing.T) {
	schemaResp := &resource.SchemaResponse{}
	(&EnvironmentResource{}).Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	attr := schemaResp.Schema.Attributes["is_edge"].(resourceschema.BoolAttribute)

	// Non-null resource state and plan: an update, not a create or destroy.
	raw := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})

	cases := []struct {
		name  string
		state types.Bool
		plan  types.Bool
		want  bool
	}{
		{"changed", types.BoolValue(false), types.BoolValue(true), true},
		{"unchanged", types.BoolValue(true), types.BoolValue(true), false},
		{"null state after import", types.BoolNull(), types.BoolValue(true), false},
		{"removed from config", types.BoolValue(true), types.BoolNull(), false},
		{"unknown plan", types.BoolValue(true), types.BoolUnknown(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.BoolRequest{
				State:       tfsdk.State{Raw: raw},
				Plan:        tfsdk.Plan{Raw: raw},
				StateValue:  tc.state,
				PlanValue:   tc.plan,
				ConfigValue: tc.plan,
			}
			resp := &planmodifier.BoolResponse{PlanValue: tc.plan}
			for _, m := range attr.PlanModifiers {
				m.PlanModifyBool(context.Background(), req, resp)
			}
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}
