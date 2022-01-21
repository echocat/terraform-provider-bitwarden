package plugin

import (
	"context"
	"fmt"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

func dataSourceItems() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceItemsRead,
		Schema: map[string]*schema.Schema{
			"organization_id": &organizationIdSchema,
			"collection_id":   &collectionIdSchema,
			"folder_id":       &folderIdSchema,
			"search": {
				Type:     schema.TypeString,
				Required: true,
			},
			"attachments_query": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &attachmentQuerySchema,
			},
			"matches": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &itemSchema,
			},
		},
	}
}

func dataSourceItemsRead(_ context.Context, d *schema.ResourceData, plainB interface{}) (diags diag.Diagnostics) {
	b := plainB.(*bitwarden.Bitwarden)
	q := bitwarden.ItemsQuery{
		Search: d.Get("search").(string),
		OnTooBroadQuery: func() {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "No limitation provided.",
				Detail:   "It is strongly recommend to provide at least one limitation of: organization_id, collection_id, folder_id. Otherwise too many items might be returned.",
			})
		},
	}

	if v, ok := d.Get("organization_id").(string); ok {
		q.OrganizationId = v
	}
	if v, ok := d.Get("collection_id").(string); ok && v != "" {
		q.CollectionId = v
	}
	if v, ok := d.Get("folder_id").(string); ok && v != "" {
		q.FolderId = v
	}

	if err := q.Attachments.Parse(d.Get("attachments_query")); err != nil {
		return diag.FromErr(err)
	}

	items, err := b.FindItems(q)
	if err != nil {
		return diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Cannot execute bitwarden command.",
			Detail:   fmt.Sprintf("Cannot execute bitwarden command: %v", err),
		}}
	}

	result := items.ToResponse()

	if err := d.Set("matches", result); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
