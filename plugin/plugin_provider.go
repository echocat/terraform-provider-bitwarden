package plugin

import (
	"context"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func (this *Plugin) provider() *schema.Provider {

	provider := schema.Provider{
		Schema: map[string]*schema.Schema{
			"executable": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("BW_EXECUTABLE", "bw"),
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"bitwarden_items": dataSourceItems(),
			"bitwarden_item":  dataSourceItem(),
		},
		ConfigureContextFunc: this.providerConfigure,
	}
	if this.BitwardenHolder == nil {
		provider.Schema["session"] = &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			DefaultFunc: schema.EnvDefaultFunc("BW_SESSION", nil),
		}
	}

	return &provider
}

func (this *Plugin) providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var result *bitwarden.Bitwarden
	if p := this.BitwardenHolder; p != nil {
		b, err := p.Bitwarden()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		return b, nil
	}

	session := d.Get("session").(string)
	executable := d.Get("executable").(string)

	result, err := bitwarden.NewBitwarden(session, executable)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	if ok, err := result.Test(); err != nil {
		return nil, diag.FromErr(err)
	} else if !ok {
		return nil, diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Bitwarden session wrong or expired.",
			Detail:   bitwarden.DetailWrongSession,
		}}
	}
	return result, nil
}
