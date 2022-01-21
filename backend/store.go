package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bhoriuchi/terraform-backend-http/go/store"
	"github.com/bhoriuchi/terraform-backend-http/go/types"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"regexp"
	"sort"
	"strings"
	"time"
)

type StoreParent interface {
	Bitwarden() (*bitwarden.Bitwarden, error)
	GetOrganizationId() string
	GetCollectionId() string
	GetFolderId() string
	GetConfig() Config
}

type Store struct {
	StoreParent
}

func (this *Store) Init() error {
	return nil
}

func (this *Store) getItem(b *bitwarden.Bitwarden, plainRef string) (*bitwarden.Item, error) {
	ref, err := NewStoreRef(plainRef)
	if err != nil {
		return nil, err
	}

	if err := b.Sync(); err != nil {
		return nil, err
	}

	if ref.ItemId != "" {
		return b.GetItem(ref.ItemId)
	}
	if ref.ItemName != "" {
		return b.FindItem(bitwarden.ItemQuery{
			Name:           ref.ItemName,
			OrganizationId: this.GetOrganizationId(),
			CollectionId:   this.GetCollectionId(),
			FolderId:       this.GetFolderId(),
		})
	}
	return nil, fmt.Errorf("%w: %s", ErrIllegalStoreRef, plainRef)
}

func (this *Store) GetState(plainRef string) (state map[string]interface{}, encrypted bool, err error) {
	b, err := this.Bitwarden()
	if err != nil {
		return nil, false, err
	}

	item, err := this.getItem(b, plainRef)
	if err == bitwarden.ErrNoSuchItem {
		return nil, false, store.ErrNotFound
	}
	if err != nil {
		return nil, false, err
	}

	aref := this.latestAttachmentReferenceOf(item)
	if aref == nil {
		return nil, false, store.ErrNotFound
	}

	attachment, err := b.GetAttachment(*item, aref.Id, false)
	if err != nil {
		return nil, false, err
	}

	state = map[string]interface{}{}
	if err := json.Unmarshal([]byte(attachment), &state); err != nil {
		return nil, false, fmt.Errorf("cannot decode state of attachment %s of item %v (%v)", aref.FileName, item.Name, item.Id)
	}

	return state, false, nil
}

func (this *Store) PutState(plainRef string, state, metadata map[string]interface{}, encrypted bool) error {
	if encrypted {
		return fmt.Errorf("encryption of states are not supported, because inside of Bitwarden it is already encrypted")
	}
	if len(metadata) > 0 {
		return fmt.Errorf("currently there are no metadata supported")
	}

	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}

	b, err := this.Bitwarden()
	if err != nil {
		return err
	}

	item, err := this.getItem(b, plainRef)
	if err != nil {
		return err
	}

	fn := this.newStateFilename()

	if err := b.CreateAttachment(*item, fn, encoded); err != nil {
		return err
	}

	var arefs timedAttachmentReferences
	arefs.extractIfPossibleFrom(&item.AttachmentReferences)

	maxHistory := this.GetConfig().GetState().GetMaxRevisions() - 1 // 1 = the item we just added
	if len(arefs) > int(maxHistory) {
		sort.Sort(sort.Reverse(arefs))

		for _, aref := range arefs[:maxHistory] {
			if err := b.DeleteAttachment(*item, *aref.ItemAttachmentReference); err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *Store) DeleteState(plainRef string) error {
	b, err := this.Bitwarden()
	if err != nil {
		return err
	}

	item, err := this.getItem(b, plainRef)
	if err != nil {
		return err
	}

	var arefs timedAttachmentReferences
	arefs.extractIfPossibleFrom(&item.AttachmentReferences)

	for _, aref := range arefs {
		if err := b.DeleteAttachment(*item, *aref.ItemAttachmentReference); err != nil {
			return err
		}
	}

	return nil
}

func (this *Store) GetLock(plainRef string) (*types.Lock, error) {
	b, err := this.Bitwarden()
	if err != nil {
		return nil, err
	}

	item, err := this.getItem(b, plainRef)
	if err != nil {
		return nil, err
	}

	for _, aref := range item.AttachmentReferences {
		if aref.FileName == lockAttachmentFileName {
			attachment, err := b.GetAttachment(*item, aref.Id, false)
			if err != nil {
				return nil, err
			}

			buf := types.Lock{}
			if err := json.Unmarshal([]byte(attachment), &buf); err != nil {
				return nil, fmt.Errorf("cannot decode lock of attachment %s of item %v (%v)", aref.FileName, item.Name, item.Id)
			}

			return &buf, nil
		}
	}
	return nil, store.ErrNotFound
}

func (this *Store) PutLock(plainRef string, lock types.Lock) error {
	b, err := this.Bitwarden()
	if err != nil {
		return err
	}

	item, err := this.getItem(b, plainRef)
	if err != nil {
		return err
	}

	var attachmentsToDelete []bitwarden.ItemAttachmentReference
	for _, aref := range item.AttachmentReferences {
		if aref.FileName == lockAttachmentFileName {
			attachmentsToDelete = append(attachmentsToDelete, aref)
		}
	}

	buf, err := json.Marshal(lock)
	if err != nil {
		return fmt.Errorf("cannot encode lock for item %v (%v)", item.Name, item.Id)
	}

	if err := b.CreateAttachment(*item, lockAttachmentFileName, buf); err != nil {
		return err
	}

	for _, aref := range attachmentsToDelete {
		if err := b.DeleteAttachment(*item, aref); err != nil {
			return err
		}
	}

	return nil
}

func (this *Store) DeleteLock(plainRef string) error {
	b, err := this.Bitwarden()
	if err != nil {
		return err
	}

	item, err := this.getItem(b, plainRef)
	if err != nil {
		return err
	}

	for _, aref := range item.AttachmentReferences {
		if aref.FileName == lockAttachmentFileName {
			if err := b.DeleteAttachment(*item, aref); err != nil {
				return err
			}
		}
	}

	return nil
}

const storeAttachmentFileTimePattern = "2006-01-02T15-04-05.000000"
const lockAttachmentFileName = "terraform.lock.json"

var stateAttachmentFileRegex = regexp.MustCompile(`^terraform-state-(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}\.\d{6})\.json$`)

func (this *Store) latestAttachmentReferenceOf(item *bitwarden.Item) (result *timedAttachmentReference) {
	if item != nil {
		var refs timedAttachmentReferences
		refs.extractIfPossibleFrom(&item.AttachmentReferences)
		sort.Sort(refs)
		if len(refs) > 0 {
			result = &refs[0]
		}
	}
	return
}

type timedAttachmentReference struct {
	time time.Time
	*bitwarden.ItemAttachmentReference
}

func (this *Store) newStateFilename() string {
	t := time.Now().UTC().Format(storeAttachmentFileTimePattern)
	return "terraform-state-" + t + ".json"
}

func (this *timedAttachmentReference) extractIfPossibleFrom(ref bitwarden.ItemAttachmentReference) bool {
	m := stateAttachmentFileRegex.FindStringSubmatch(ref.FileName)
	if m == nil {
		return false
	}

	parsed, err := time.Parse(storeAttachmentFileTimePattern+"Z", m[1]+"Z")
	if err != nil {
		return false
	}
	this.time = parsed
	this.ItemAttachmentReference = &ref

	return true
}

type timedAttachmentReferences []timedAttachmentReference

func (this *timedAttachmentReferences) extractIfPossibleFrom(refs *bitwarden.ItemAttachmentReferences) {
	var bufs timedAttachmentReferences
	if refs != nil {
		for _, ref := range *refs {
			var buf timedAttachmentReference
			if buf.extractIfPossibleFrom(ref) {
				bufs = append(bufs, buf)
			}
		}
	}
	*this = bufs
}

func (a timedAttachmentReferences) Less(i, j int) bool {
	return a[i].time.After(a[j].time)
}

func (a timedAttachmentReferences) Len() int      { return len(a) }
func (a timedAttachmentReferences) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func NewStoreRef(plain string) (StoreRef, error) {
	var buf StoreRef
	if err := buf.Set(plain); err != nil {
		return StoreRef{}, nil
	}
	return buf, nil
}

var (
	ErrIllegalStoreRef = errors.New("illegal store ref")
)

type StoreRef struct {
	ItemId   string
	ItemName string
}

func (this *StoreRef) Set(plain string) error {
	var buf StoreRef

	if plain != "" {
		parts := strings.SplitN(plain, ":", 2)
		if len(parts) < 2 {
			return fmt.Errorf("%w: %s", ErrIllegalStoreRef, plain)
		}

		switch parts[0] {
		case "id":
			buf.ItemId = parts[1]
		case "name":
			buf.ItemName = parts[1]
		default:
			return fmt.Errorf("%w: %s", ErrIllegalStoreRef, plain)
		}
	}

	*this = buf
	return nil
}

func (this StoreRef) String() string {
	if v := this.ItemId; v != "" {
		return "id:" + v
	}
	if v := this.ItemName; v != "" {
		return "name:" + v
	}
	return ""
}
