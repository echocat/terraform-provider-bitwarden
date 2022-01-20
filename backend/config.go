package backend

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"os"
	"path/filepath"
)

func NewConfig() *Config {
	return &Config{
		Variables: NewConfigVariables(),
		State:     NewConfigState(),
		Bitwarden: NewConfigBitwarden(),
		Terraform: NewConfigTerraform(),
		Backend:   NewConfigBackend(),
	}
}

type Config struct {
	Variables ConfigVariables  `hcl:"variable,block"`
	State     *ConfigState     `hcl:"state,block"`
	Bitwarden *ConfigBitwarden `hcl:"bitwarden,block"`
	Terraform *ConfigTerraform `hcl:"terraform,block"`
	Backend   *ConfigBackend   `hcl:"backend,block"`
}

func (this Config) Validate() error {
	if err := this.Variables.Validate(); err != nil {
		return err
	}
	if v := this.State; v != nil {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if v := this.Bitwarden; v != nil {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if v := this.Terraform; v != nil {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if v := this.Backend; v != nil {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (this *Config) ReadFile(fn string, ctx *hcl.EvalContext) error {
	buf := NewConfig()
	if err := hclsimple.DecodeFile(fn, ctx, buf); err != nil {
		return fmt.Errorf("cannot read configuration file %s: %w", fn, err)
	}
	if buf.State == nil {
		buf.State = NewConfigState()
	}
	if buf.Bitwarden == nil {
		buf.Bitwarden = NewConfigBitwarden()
	}
	if buf.Terraform == nil {
		buf.Terraform = NewConfigTerraform()
	}
	if buf.Backend == nil {
		buf.Backend = NewConfigBackend()
	}

	*this = *buf
	return nil
}

func (this *Config) Read(ctx *hcl.EvalContext) error {
	var local, user Config

	userDir, err := os.UserConfigDir()
	if err == nil {
		userFile := filepath.Join(userDir, "terraform-backend-bitwarden", "config.hcl")
		if err := user.ReadFile(userFile, ctx); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	childCtx := ctx.NewChild()
	childCtx.Variables = map[string]cty.Value{
		"configs": cty.ObjectVal(map[string]cty.Value{
			"user": user.ToValue(),
		}),
	}
	if err := local.ReadFile(".backend-bitwarden.hcl", childCtx); err != nil && !os.IsNotExist(err) {
		return err
	}

	*this = local.Merge(user)
	return nil
}

func (this Config) Merge(with Config) Config {
	return Config{
		Variables: this.Variables.Merge(with.Variables),
		State:     MergeConfigState(this.State, with.State),
		Bitwarden: MergeConfigBitwarden(this.Bitwarden, with.Bitwarden),
		Terraform: MergeConfigTerraform(this.Terraform, with.Terraform),
		Backend:   MergeConfigBackend(this.Backend, with.Backend),
	}
}

func (this Config) ToValue() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"variables": this.Variables.ToValue(),
		"state":     ConfigStateToValue(this.State),
		"bitwarden": ConfigBitwardenToValue(this.Bitwarden),
		"terraform": ConfigTerraformToValue(this.Terraform),
		"backend":   ConfigBackendToValue(this.Backend),
	})
}

func (this Config) GetState() ConfigState {
	if v := this.State; v != nil {
		return *v
	}
	return *NewConfigState()
}

func (this Config) GetBitwarden() ConfigBitwarden {
	if v := this.Bitwarden; v != nil {
		return *v
	}
	return *NewConfigBitwarden()
}

func (this Config) GetTerraform() ConfigTerraform {
	if v := this.Terraform; v != nil {
		return *v
	}
	return *NewConfigTerraform()
}

func (this Config) GetBackend() ConfigBackend {
	if v := this.Backend; v != nil {
		return *v
	}
	return *NewConfigBackend()
}
