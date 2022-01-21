package plugin

import (
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	itemSchema = schema.Resource{
		Schema: map[string]*schema.Schema{
			"organization_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"folder_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"id": {
				Type:     schema.TypeString,
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
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"username": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"password": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
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

	attachmentQuerySchema = schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"filename_matches": {
				Type:     schema.TypeString,
				Required: true,
			},
			"base64_encode": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"unique": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}

	organizationIdSchema = schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		ValidateDiagFunc: func(v interface{}, path cty.Path) (diags diag.Diagnostics) {
			if vStr, ok := v.(string); ok && vStr != "" {
				if _, err := uuid.ParseUUID(vStr); err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Illegal bitwarden organization_id.",
						Detail:   fmt.Sprintf("Illegal bitwarden organization_id: %v", vStr),
					})
				}
			}
			return
		},
	}
	collectionIdSchema = schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		ValidateDiagFunc: func(v interface{}, path cty.Path) (diags diag.Diagnostics) {
			if vStr, ok := v.(string); ok && vStr != "" {
				if _, err := uuid.ParseUUID(vStr); err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Illegal bitwarden collection_id.",
						Detail:   fmt.Sprintf("Illegal bitwarden collection_id: %v", vStr),
					})
				}
			}
			return
		},
	}
	folderIdSchema = schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		ValidateDiagFunc: func(v interface{}, path cty.Path) (diags diag.Diagnostics) {
			if vStr, ok := v.(string); ok && vStr != "" {
				if _, err := uuid.ParseUUID(vStr); err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Illegal bitwarden folder_id.",
						Detail:   fmt.Sprintf("Illegal bitwarden folder_id: %v", vStr),
					})
				}
			}
			return
		},
	}
	idSchema = schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		ValidateDiagFunc: func(v interface{}, path cty.Path) (diags diag.Diagnostics) {
			if vStr, ok := v.(string); ok && vStr != "" {
				if _, err := uuid.ParseUUID(vStr); err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Illegal bitwarden id.",
						Detail:   fmt.Sprintf("Illegal bitwarden id: %v", vStr),
					})
				}
			}
			return
		},
	}
)
