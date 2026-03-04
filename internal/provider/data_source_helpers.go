package provider

import (
	"encoding/json"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func setProviderClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *sdkclient.Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*sdkclient.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", "Expected *sdkclient.Client")
		return nil
	}
	return client
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

type categoriesModel struct {
	TotalCount types.Int64  `tfsdk:"total_count"`
	DataJSON   types.String `tfsdk:"data_json"`
}
