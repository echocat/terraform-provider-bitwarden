package backend

import (
	"fmt"
	log "github.com/echocat/slf4g"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"
	"regexp"
)

var (
	varNameRegex = regexp.MustCompile("^[a-z_]+$")
)

type ConfigVariable struct {
	Label string `hcl:"label,label"`

	ItemId         string `hcl:"item_id,optional"`
	OrganizationId string `hcl:"organization_id,optional"`
	CollectionId   string `hcl:"collection_id,optional"`
	FolderId       string `hcl:"folder_id,optional"`
	Name           string `hcl:"name,optional"`
	Field          string `hcl:"field,optional"`

	Ref string `hcl:"ref,optional"`
}

func (this ConfigVariable) Validate() error {
	if this.Label == "" {
		return fmt.Errorf("vairable without label")
	}
	if !varNameRegex.MatchString(this.Label) {
		return fmt.Errorf("illegal variable label: '%s'", this.Label)
	}

	if _, err := uuid.ParseUUID(this.ItemId); this.ItemId != "" && err != nil {
		return fmt.Errorf("%s: illegal item_id: '%s'", this.Label, this.ItemId)
	}
	if _, err := uuid.ParseUUID(this.OrganizationId); this.OrganizationId != "" && err != nil {
		return fmt.Errorf("%s: illegal organization_id: '%s'", this.Label, this.OrganizationId)
	}
	if _, err := uuid.ParseUUID(this.CollectionId); this.CollectionId != "" && err != nil {
		return fmt.Errorf("%s: illegal collection_id: '%s'", this.Label, this.CollectionId)
	}
	if _, err := uuid.ParseUUID(this.FolderId); this.FolderId != "" && err != nil {
		return fmt.Errorf("%s: illegal folder_id: '%s'", this.Label, this.FolderId)
	}
	if this.Ref != "" && !varNameRegex.MatchString(this.Ref) {
		return fmt.Errorf("%s: illegal ref: '%s'", this.Label, this.Ref)
	}

	if this.Name != "" && this.ItemId != "" && this.Ref != "" {
		return fmt.Errorf("%s: attribute name, item_id and ref cannot be used together", this.Label)
	}
	if this.Name == "" && this.ItemId == "" && this.Ref == "" {
		return fmt.Errorf("%s: one attribute of name, item_id or ref is required", this.Label)
	}

	return nil
}

func (this ConfigVariable) ToValue() (string, cty.Value) {
	return this.Label, cty.ObjectVal(map[string]cty.Value{
		"item_id":         cty.StringVal(this.ItemId),
		"organization_id": cty.StringVal(this.OrganizationId),
		"collection_id":   cty.StringVal(this.CollectionId),
		"folder_id":       cty.StringVal(this.FolderId),
		"name":            cty.StringVal(this.Name),
		"field":           cty.StringVal(this.Field),
		"ref":             cty.StringVal(this.Ref),
	})
}

func (this ConfigVariable) Resolve(using *bitwarden.Bitwarden, refs ConfigVariables) (string, error) {
	result, err := this.resolve(using, refs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", this.Label, err)
	}
	return result, nil
}

func (this ConfigVariable) resolve(using *bitwarden.Bitwarden, refs ConfigVariables) (_ string, err error) {
	if ref := this.Ref; ref != "" {
		refVar, ok := refs.Lookup(ref)
		if !ok {
			return "", fmt.Errorf("ref '%s' cannot be resolved", ref)
		}
		return refVar.resolve(using, refs)
	}

	var item *bitwarden.Item
	if this.ItemId != "" {
		if item, err = using.GetItem(this.ItemId); err != nil {
			return "", fmt.Errorf("%s: %w", this.Label, err)
		}
	} else {
		if item, err = using.FindItem(this.toItemQuery()); err != nil {
			return "", fmt.Errorf("%s: %w", this.Label, err)
		}
	}

	switch this.Field {
	case "", "password":
		return item.Login.Password, nil
	case "username":
		return item.Login.Username, nil
	case "totp":
		return item.Login.Totp, nil
	case "uri":
		if len(item.Login.Uris) <= 0 {
			return "", nil
		}
		return item.Login.Uris[0].Uri, nil
	case "organization_id":
		if item.OrganizationId == nil {
			return "", nil
		}
		return *item.OrganizationId, nil
	case "folder_id":
		if item.FolderId == nil {
			return "", nil
		}
		return *item.FolderId, nil
	case "collection_id":
		if len(item.CollectionIds) <= 0 {
			return "", nil
		}
		return item.CollectionIds[0], nil
	case "name":
		return item.Name, nil
	case "id":
		return item.Id, nil
	case "favorite":
		return item.Id, nil
	default:
		if v, ok := item.Fields.Lookup(this.Field); ok {
			return v.Value, nil
		}
		return "", fmt.Errorf("unknown field: %s", this.Field)
	}
}

func (this ConfigVariable) toItemQuery() bitwarden.ItemQuery {
	return bitwarden.ItemQuery{
		Name:           this.Name,
		OrganizationId: this.OrganizationId,
		CollectionId:   this.CollectionId,
		FolderId:       this.FolderId,
		OnTooBroadQuery: func() {
			log.With("variable", this.Label).
				With("itemName", this.Name).
				Warn("It is strongly recommend to provide at least one limitation of: organization_id, collection_id, folder_id. Otherwise too many items might be returned.")
		},
	}
}

func NewConfigVariables() ConfigVariables {
	return nil
}

type ConfigVariables []ConfigVariable

func (this ConfigVariables) Validate() error {
	for _, v := range this {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (this ConfigVariables) ToValue() cty.Value {
	vals := make(map[string]cty.Value, len(this))
	for _, v := range this {
		k, nv := v.ToValue()
		vals[k] = nv
	}
	return cty.ObjectVal(vals)
}

func (this ConfigVariables) Merge(with ConfigVariables) ConfigVariables {
	result := make([]ConfigVariable, len(this)+len(with))
	knownVars := make(map[string]struct{}, len(result))

	i := 0
	for _, v := range this {
		if _, exists := knownVars[v.Label]; !exists {
			result[i] = v
			i++
		}
	}
	for _, v := range with {
		if _, exists := knownVars[v.Label]; !exists {
			result[i] = v
			i++
		}
	}
	return result[:i]
}

func (this ConfigVariables) Resolve(using *bitwarden.Bitwarden) (map[string]string, error) {
	result := make(map[string]string, len(this))
	for _, v := range this {
		nv, err := v.Resolve(using, this)
		if err != nil {
			return nil, err
		}
		result[v.Label] = nv
	}
	return result, nil
}

func (this ConfigVariables) Lookup(label string) (ConfigVariable, bool) {
	for _, v := range this {
		if v.Label == label {
			return v, true
		}
	}
	return ConfigVariable{}, false
}
