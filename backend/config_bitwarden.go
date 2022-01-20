package backend

import (
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/zclconf/go-cty/cty"
)

func NewConfigBitwarden() *ConfigBitwarden {
	return &ConfigBitwarden{}
}

type ConfigBitwarden struct {
	Executable       string `hcl:"executable,optional"`
	Session          string `hcl:"session,optional"`
	UnlockIfRequired *bool  `hcl:"unlock_if_required,optional"`
}

func (this ConfigBitwarden) NewBitwarden() (*bitwarden.Bitwarden, error) {
	return bitwarden.NewBitwarden(this.Session, this.Executable)
}

func (this ConfigBitwarden) Validate() error {
	return nil
}

func (this ConfigBitwarden) ToValue() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"executable":         cty.StringVal(this.Executable),
		"session":            cty.StringVal(this.Session),
		"unlock_if_required": cty.BoolVal(this.UnlockIfRequired == nil || *this.UnlockIfRequired),
	})
}

func ConfigBitwardenToValue(v *ConfigBitwarden) cty.Value {
	if v == nil {
		return NewConfigBitwarden().ToValue()
	}
	return v.ToValue()
}

func (this ConfigBitwarden) Merge(what ConfigBitwarden) ConfigBitwarden {
	executable := this.Executable
	if executable == "" {
		executable = what.Executable
	}
	session := this.Session
	if session == "" {
		session = what.Session
	}
	unlockIfRequired := this.UnlockIfRequired
	if unlockIfRequired == nil {
		unlockIfRequired = what.UnlockIfRequired
	}
	return ConfigBitwarden{
		Executable:       executable,
		Session:          session,
		UnlockIfRequired: unlockIfRequired,
	}
}

func (this ConfigBitwarden) IsUnlockIfRequired() bool {
	if v := this.UnlockIfRequired; v != nil {
		return *v
	}
	return true
}

func MergeConfigBitwarden(a, b *ConfigBitwarden) *ConfigBitwarden {
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

func (this ConfigBitwarden) GetExecutable() string {
	if v := this.Executable; v != "" {
		return v
	}
	return "bw"
}

func (this ConfigBitwarden) GetSession() string {
	return this.Session
}
