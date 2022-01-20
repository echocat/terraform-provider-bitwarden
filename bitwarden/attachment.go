package bitwarden

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Attachment []byte

func (this Attachment) String() string {
	return base64.StdEncoding.EncodeToString(this)
}

func (this Attachment) ToReader() io.Reader {
	return bytes.NewReader(this)
}

func (this Attachment) ToTempFile(name string) (*AttachmentTempFile, error) {
	dir, err := os.MkdirTemp("", "terraform-provider-bitwarden-attachment-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temporary file for attachment: %w", err)
	}
	fn := filepath.Join(dir, name)
	if err := ioutil.WriteFile(fn, this, 0644); err != nil {
		return nil, fmt.Errorf("cannot create temporary file for attachment: %w", err)
	}
	return &AttachmentTempFile{
		Name: fn,
	}, nil
}

type AttachmentTempFile struct {
	Name string
}

func (this *AttachmentTempFile) Close() error {
	if err := os.RemoveAll(filepath.Base(this.Name)); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (this AttachmentTempFile) String() string {
	return this.Name
}
