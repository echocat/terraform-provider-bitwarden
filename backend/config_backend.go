package backend

import (
	"github.com/zclconf/go-cty/cty"
)

func NewConfigBackend() *ConfigBackend {
	return &ConfigBackend{}
}

type ConfigBackend struct {
	Port uint16 `hcl:"port,optional"`
}

func (this ConfigBackend) Validate() error {
	return nil
}

func (this ConfigBackend) ToValue() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"port": cty.NumberUIntVal(uint64(this.Port)),
	})
}

func ConfigBackendToValue(v *ConfigBackend) cty.Value {
	if v == nil {
		return NewConfigBackend().ToValue()
	}
	return v.ToValue()
}

func (this ConfigBackend) Merge(what ConfigBackend) ConfigBackend {
	port := this.Port
	if port == 0 {
		port = what.Port
	}
	return ConfigBackend{
		Port: port,
	}
}

func MergeConfigBackend(a, b *ConfigBackend) *ConfigBackend {
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

func (this ConfigBackend) GetPort() uint16 {
	if v := this.Port; v > 0 {
		return v
	}
	return 26394
}
