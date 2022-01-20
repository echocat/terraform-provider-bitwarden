package backend

import (
	"fmt"
	log "github.com/echocat/slf4g"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"
)

func NewConfigState() *ConfigState {
	return &ConfigState{}
}

type ConfigState struct {
	MaxRevisions uint16 `hcl:"max_revisions,optional"`

	ItemId         string `hcl:"item_id,optional"`
	OrganizationId string `hcl:"organization_id,optional"`
	CollectionId   string `hcl:"collection_id,optional"`
	FolderId       string `hcl:"folder_id,optional"`
	ItemName       string `hcl:"item_name,optional"`
}

func (this ConfigState) Validate() error {
	if _, err := uuid.ParseUUID(this.ItemId); this.ItemId != "" && err != nil {
		return fmt.Errorf("state: illegal item_id: '%s'", this.ItemId)
	}
	if _, err := uuid.ParseUUID(this.OrganizationId); this.OrganizationId != "" && err != nil {
		return fmt.Errorf("state: illegal organization_id: '%s'", this.OrganizationId)
	}
	if _, err := uuid.ParseUUID(this.CollectionId); this.CollectionId != "" && err != nil {
		return fmt.Errorf("state: illegal collection_id: '%s'", this.CollectionId)
	}
	if _, err := uuid.ParseUUID(this.FolderId); this.FolderId != "" && err != nil {
		return fmt.Errorf("state: illegal folder_id: '%s'", this.FolderId)
	}

	if this.ItemName != "" && this.ItemId != "" {
		return fmt.Errorf("state: attribute name and item_id cannot be used together")
	}
	if this.ItemName == "" && this.ItemId == "" {
		return fmt.Errorf("state: one attribute of item_name or item_id ref is required")
	}

	return nil
}

func (this ConfigState) toItemQuery() bitwarden.ItemQuery {
	return bitwarden.ItemQuery{
		Name:           this.ItemName,
		OrganizationId: this.OrganizationId,
		CollectionId:   this.CollectionId,
		FolderId:       this.FolderId,
		OnTooBroadQuery: func() {
			log.With("itemName", this.ItemName).
				Warn("It is strongly recommend to provide at least one limitation of: organization_id, collection_id, folder_id. Otherwise too many items might be returned.")
		},
	}
}

func (this ConfigState) ToValue() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"max_revisions": cty.NumberUIntVal(uint64(this.MaxRevisions)),

		"item_id":         cty.StringVal(this.ItemId),
		"organization_id": cty.StringVal(this.OrganizationId),
		"collection_id":   cty.StringVal(this.CollectionId),
		"folder_id":       cty.StringVal(this.FolderId),
		"item_name":       cty.StringVal(this.ItemName),
	})
}

func ConfigStateToValue(v *ConfigState) cty.Value {
	if v == nil {
		return NewConfigState().ToValue()
	}
	return v.ToValue()
}

func (this ConfigState) Merge(with ConfigState) ConfigState {
	maxRevisions := this.MaxRevisions
	if maxRevisions == 0 {
		maxRevisions = with.MaxRevisions
	}
	itemId := this.ItemId
	if itemId == "" {
		itemId = with.ItemId
	}
	organizationId := this.OrganizationId
	if organizationId == "" {
		organizationId = with.OrganizationId
	}
	collectionId := this.CollectionId
	if collectionId == "" {
		collectionId = with.CollectionId
	}
	folderId := this.FolderId
	if folderId == "" {
		folderId = with.FolderId
	}
	itemName := this.ItemName
	if itemName == "" {
		itemName = with.ItemName
	}
	return ConfigState{
		MaxRevisions:   maxRevisions,
		ItemId:         itemId,
		OrganizationId: organizationId,
		CollectionId:   collectionId,
		FolderId:       folderId,
		ItemName:       itemName,
	}
}

func MergeConfigState(a, b *ConfigState) *ConfigState {
	if a != nil && b != nil {
		nv := a.Merge(*b)
		return &nv
	}
	if a != nil {
		return a
	}
	if b != nil {
		return b
	}
	return nil
}

func (this ConfigState) GetMaxRevisions() uint16 {
	if v := this.MaxRevisions; v > 0 {
		return v
	}
	return 2
}
