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

func dataSourceItem() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceItemRead,
		Schema: map[string]*schema.Schema{
			"organization_id": &organizationIdSchema,
			"collection_id":   &collectionIdSchema,
			"folder_id":       &folderIdSchema,
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"attachments_query": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &attachmentQuerySchema,
			},
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"reprompt": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"username": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"password": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"uris": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"collection_ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"attachments": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceItemRead(_ context.Context, d *schema.ResourceData, plainB interface{}) (diags diag.Diagnostics) {
	b := plainB.(*bitwarden.Bitwarden)
	q := bitwarden.ItemQuery{
		Name: d.Get("name").(string),
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

	var attachmentQueries bitwarden.ItemAttachmentQueries
	if err := attachmentQueries.Parse(d.Get("attachments_query")); err != nil {
		return diag.FromErr(err)
	}

	item, err := b.FindItem(q)
	if err == bitwarden.ErrNoSuchItem {
		return diag.Diagnostics{diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "No such entry.",
			Detail:   fmt.Sprintf("Cannot find entry named '%v'.", q.Name),
		}}
	}
	if err == bitwarden.ErrItemNotUnique {
		return diag.Diagnostics{diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "No unique entry.",
			Detail:   fmt.Sprintf("Found more than one item matching name '%v'.", q.Name),
		}}
	}
	if err != nil {
		return diag.FromErr(err)
	}
	for k, v := range item.ToResponse() {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return
}
