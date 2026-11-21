package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// containerHostConfigAttributes are the container settings Arcane takes in
// hostConfig and healthcheck. Containers are immutable, so every one forces a
// new container.
func containerHostConfigAttributes() map[string]resourceschema.Attribute {
	replaceList := []planmodifier.List{listplanmodifier.RequiresReplace()}
	replaceBool := []planmodifier.Bool{boolplanmodifier.RequiresReplace()}
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	return map[string]resourceschema.Attribute{
		"cap_add":           resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Linux capabilities to add, e.g. NET_ADMIN.", PlanModifiers: replaceList},
		"cap_drop":          resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Linux capabilities to drop, e.g. ALL.", PlanModifiers: replaceList},
		"readonly_rootfs":   resourceschema.BoolAttribute{Optional: true, Description: "Mount the container's root filesystem read-only.", PlanModifiers: replaceBool},
		"publish_all_ports": resourceschema.BoolAttribute{Optional: true, Description: "Publish every exposed port on a random host port.", PlanModifiers: replaceBool},
		"memory_swap":       resourceschema.Int64Attribute{Optional: true, Description: "Memory plus swap limit in bytes; -1 for unlimited swap.", PlanModifiers: replaceInt},
		"cpu_shares":        resourceschema.Int64Attribute{Optional: true, Description: "Relative CPU weight.", PlanModifiers: replaceInt},
		"network_mode": resourceschema.StringAttribute{
			Optional:      true,
			Description:   "Network mode: bridge, host, none, container:<name> or a network name.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"restart_max_retries": resourceschema.Int64Attribute{
			Optional:      true,
			Description:   "Maximum restarts with restart_policy = \"on-failure\".",
			PlanModifiers: replaceInt,
		},
		"mounts": resourceschema.ListNestedAttribute{
			Optional:      true,
			Description:   "Mounts, in addition to `volumes`.",
			PlanModifiers: replaceList,
			NestedObject: resourceschema.NestedAttributeObject{
				Attributes: map[string]resourceschema.Attribute{
					"type":      resourceschema.StringAttribute{Required: true, Description: "bind, volume or tmpfs."},
					"source":    resourceschema.StringAttribute{Required: true, Description: "Host path or volume name."},
					"target":    resourceschema.StringAttribute{Required: true, Description: "Path in the container."},
					"read_only": resourceschema.BoolAttribute{Optional: true, Description: "Mount read-only."},
				},
			},
		},
		"healthcheck": resourceschema.SingleNestedAttribute{
			Optional:      true,
			Description:   "Overrides the image healthcheck. Durations are in seconds.",
			PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
			Attributes: map[string]resourceschema.Attribute{
				"test": resourceschema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Check command, e.g. [\"CMD-SHELL\", \"curl -f http://localhost/ || exit 1\"]; [\"NONE\"] disables the image healthcheck.",
				},
				"interval":       resourceschema.Int64Attribute{Optional: true, Description: "Time between checks."},
				"timeout":        resourceschema.Int64Attribute{Optional: true, Description: "Time a check may take."},
				"retries":        resourceschema.Int64Attribute{Optional: true, Description: "Consecutive failures before unhealthy."},
				"start_period":   resourceschema.Int64Attribute{Optional: true, Description: "Start-up time during which failures do not count."},
				"start_interval": resourceschema.Int64Attribute{Optional: true, Description: "Time between checks during the start period."},
			},
		},
	}
}

type containerMountModel struct {
	Type     types.String `tfsdk:"type"`
	Source   types.String `tfsdk:"source"`
	Target   types.String `tfsdk:"target"`
	ReadOnly types.Bool   `tfsdk:"read_only"`
}

type containerHealthcheckModel struct {
	Test          types.List  `tfsdk:"test"`
	Interval      types.Int64 `tfsdk:"interval"`
	Timeout       types.Int64 `tfsdk:"timeout"`
	Retries       types.Int64 `tfsdk:"retries"`
	StartPeriod   types.Int64 `tfsdk:"start_period"`
	StartInterval types.Int64 `tfsdk:"start_interval"`
}

func optionalInt64(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// applyContainerHostConfig adds the hostConfig and healthcheck settings of
// the plan to the create request.
func applyContainerHostConfig(ctx context.Context, plan containerModel, body *sdkclient.ContainerCreateRequest) {
	hc := &sdkclient.ContainerHostConfigCreate{
		CapAdd:          listToStrings(ctx, plan.CapAdd),
		CapDrop:         listToStrings(ctx, plan.CapDrop),
		ReadonlyRootfs:  optionalBool(plan.ReadonlyRootfs),
		PublishAllPorts: optionalBool(plan.PublishAllPorts),
		MemorySwap:      optionalInt64(plan.MemorySwap),
		CPUShares:       optionalInt64(plan.CPUShares),
		NetworkMode:     optionalString(plan.NetworkMode),
	}
	if r := optionalInt64(plan.RestartMaxRetries); r != nil {
		// hostConfig.restartPolicy replaces the top-level restartPolicy, so
		// carry the policy name over with the retry count.
		hc.RestartPolicy = &sdkclient.ContainerRestartPolicyCreate{Name: plan.RestartPolicy.ValueString(), MaximumRetryCount: *r}
	}
	for _, m := range plan.Mounts {
		hc.Mounts = append(hc.Mounts, sdkclient.ContainerMountCreate{
			Type:     m.Type.ValueString(),
			Source:   m.Source.ValueString(),
			Target:   m.Target.ValueString(),
			ReadOnly: boolValue(m.ReadOnly),
		})
	}
	if !hc.Empty() {
		body.HostConfig = hc
	}
	if h := plan.Healthcheck; h != nil {
		body.Healthcheck = &sdkclient.ContainerHealthcheckCreate{
			Test:          listToStrings(ctx, h.Test),
			Interval:      optionalInt64(h.Interval),
			Timeout:       optionalInt64(h.Timeout),
			Retries:       optionalInt64(h.Retries),
			StartPeriod:   optionalInt64(h.StartPeriod),
			StartInterval: optionalInt64(h.StartInterval),
		}
	}
}
