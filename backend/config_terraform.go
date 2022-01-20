package backend

import (
	"github.com/echocat/terraform-provider-bitwarden/utils"
	"github.com/zclconf/go-cty/cty"
)

func NewConfigTerraform() *ConfigTerraform {
	return &ConfigTerraform{}
}

type ConfigTerraform struct {
	Executable utils.Executable `hcl:"executable,optional"`
}

func (this ConfigTerraform) Validate() error {
	return nil
}

func (this ConfigTerraform) ToValue() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"executable": cty.StringVal(this.Executable.String()),
	})
}

func ConfigTerraformToValue(v *ConfigTerraform) cty.Value {
	if v == nil {
		return NewConfigTerraform().ToValue()
	}
	return v.ToValue()
}

func (this ConfigTerraform) Merge(what ConfigTerraform) ConfigTerraform {
	executable := this.Executable
	if executable == "" {
		executable = what.Executable
	}
	return ConfigTerraform{
		Executable: executable,
	}
}

func MergeConfigTerraform(a, b *ConfigTerraform) *ConfigTerraform {
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

func (this ConfigTerraform) GetExecutable() utils.Executable {
	if v := this.Executable; v != "" {
		return v
	}
	return "terraform"
}
