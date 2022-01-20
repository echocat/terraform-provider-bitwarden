package bitwarden

import (
	"fmt"
	"reflect"
	"regexp"
	"time"
)

type Item struct {
	Object               string                   `json:"object"`
	Id                   string                   `json:"id"`
	OrganizationId       *string                  `json:"organizationId"`
	FolderId             *string                  `json:"folderId"`
	Type                 int                      `json:"type"`
	Reprompt             int                      `json:"reprompt"`
	Name                 string                   `json:"name"`
	Favorite             bool                     `json:"favorite"`
	Fields               ItemFields               `json:"fields"`
	Login                ItemLogin                `json:"login"`
	CollectionIds        []string                 `json:"collectionIds"`
	AttachmentReferences ItemAttachmentReferences `json:"attachments"`
	ResolvedAttachments  ItemAttachments          `json:"-"`
	RevisionDate         *time.Time               `json:"revisionDate"`
}

func (this *Item) ResolveAttachments(by ItemAttachmentQueries, using *Bitwarden) error {
	resolved, err := using.GetAttachments(*this, by)
	if err != nil {
		return err
	}
	this.ResolvedAttachments = resolved
	return nil
}

func (this Item) ToResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":              this.Id,
		"organization_id": this.OrganizationId,
		"folder_id":       this.FolderId,
		"type":            this.Type,
		"reprompt":        this.Reprompt,
		"name":            this.Name,
		"username":        this.Login.Username,
		"password":        this.Login.Password,
		"uris":            this.Login.Uris.ToResponse(),
		"collection_ids":  this.CollectionIds,
		"attachments":     this.ResolvedAttachments,
	}
}

type ItemQuery struct {
	Name           string
	OrganizationId string
	CollectionId   string
	FolderId       string

	Attachments ItemAttachmentQueries

	OnTooBroadQuery func()
}

type Items []Item

func (this Items) ToResponse() []map[string]interface{} {
	result := make([]map[string]interface{}, len(this))

	for i, item := range this {
		result[i] = item.ToResponse()
	}

	return result
}

type ItemsQuery struct {
	Search         string
	OrganizationId string
	CollectionId   string
	FolderId       string

	Attachments ItemAttachmentQueries

	OnTooBroadQuery func()
}

type ItemLogin struct {
	Uris     ItemLoginUris `json:"uris"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	Totp     string        `json:"totp"`
}

type ItemLoginUri struct {
	Match string `json:"match"`
	Uri   string `json:"uri"`
}

type ItemLoginUris []ItemLoginUri

func (this ItemLoginUris) ToResponse() []string {
	uris := make([]string, len(this))
	for i, uri := range this {
		uris[i] = uri.Uri
	}
	return uris
}

type ItemAttachmentReference struct {
	Id       string `json:"id"`
	FileName string `json:"fileName"`
	Size     string `json:"size"`
	Url      string `json:"url"`
}

type ItemAttachmentReferences []ItemAttachmentReference

type ItemAttachments map[string]string

type ItemAttachmentQuery struct {
	Name            string
	FilenameMatches *regexp.Regexp
	Base64Encode    bool
	Unique          bool
}

func (this *ItemAttachmentQuery) Parse(plain interface{}) error {
	switch v := plain.(type) {
	case nil:
		return this.parse(nil)
	case map[string]interface{}:
		return this.parse(v)
	case *map[string]interface{}:
		return this.parse(*v)
	default:
		return fmt.Errorf("cannot parse (%v) %+v as item attachment query", reflect.TypeOf(plain), plain)
	}
}

func (this *ItemAttachmentQuery) parse(plain map[string]interface{}) error {
	if plain == nil {
		*this = ItemAttachmentQuery{}
		return nil
	}
	for k, v := range plain {
		var parser func(plain interface{}) error
		switch k {
		case "name":
			parser = this.parseName
		case "filename_matches":
			parser = this.parseFilenameMatches
		case "base64_encode":
			parser = this.parseBase64Encode
		case "unique":
			parser = this.parseUnique
		}
		if parser != nil {
			if err := parser(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *ItemAttachmentQuery) parseName(plain interface{}) error {
	if plain == nil {
		return fmt.Errorf("empty attachment_query.name provided")
	}
	if v, ok := plain.(string); ok {
		if v == "" {
			return fmt.Errorf("empty attachment_query.name provided")
		}
		this.Name = v
		return nil
	}
	return fmt.Errorf("illegal attachment_query.name provided: %+v", plain)
}

func (this *ItemAttachmentQuery) parseFilenameMatches(plain interface{}) error {
	if plain == nil {
		return fmt.Errorf("empty attachment_query.filename_matches provided")
	}
	if v, ok := plain.(string); ok {
		if v == "" {
			return fmt.Errorf("empty attachment_query.filename_matches provided")
		}
		compiled, err := regexp.Compile(v)
		if err != nil {
			return fmt.Errorf("illegal attachment_query.filename_matches regex provided: %+v", plain)
		}
		this.FilenameMatches = compiled
		return nil
	}
	return fmt.Errorf("illegal attachment_query.filename_matches provided: %+v", plain)

}

func (this *ItemAttachmentQuery) parseBase64Encode(plain interface{}) error {
	if plain == nil {
		this.Base64Encode = false
		return nil
	}
	if v, ok := plain.(bool); ok {
		this.Base64Encode = v
		return nil
	}
	return fmt.Errorf("illegal attachment_query.base64_encode provided: %+v", plain)
}

func (this *ItemAttachmentQuery) parseUnique(plain interface{}) error {
	if plain == nil {
		this.Unique = false
		return nil
	}
	if v, ok := plain.(bool); ok {
		this.Unique = v
		return nil
	}
	return fmt.Errorf("illegal attachment_query.unique provided: %+v", plain)
}

type ItemAttachmentQueries []ItemAttachmentQuery

func (this *ItemAttachmentQueries) Parse(plain interface{}) error {
	switch v := plain.(type) {
	case nil:
		return this.parse(nil)
	case []interface{}:
		return this.parse(v)
	case *[]interface{}:
		return this.parse(*v)
	default:
		return fmt.Errorf("cannot parse (%v) %+v as item attachment queries", reflect.TypeOf(plain), plain)
	}
}

func (this *ItemAttachmentQueries) parse(plain []interface{}) error {
	if plain == nil {
		*this = ItemAttachmentQueries{}
		return nil
	}
	buf := make(ItemAttachmentQueries, len(plain))
	for i, v := range plain {
		var nv ItemAttachmentQuery
		if err := nv.Parse(v); err != nil {
			return err
		}
		buf[i] = nv
	}
	*this = buf
	return nil
}

type ItemField struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Type     uint8  `json:"type"`
	LinkedId string `json:"linkedId"`
}

type ItemFields []ItemField

func (this ItemFields) Lookup(name string) (ItemField, bool) {
	for _, v := range this {
		if v.Name == name {
			return v, true
		}
	}
	return ItemField{}, false
}
