package plugin

import (
	"context"
	"encoding/json"
	log "github.com/echocat/slf4g"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"strings"
)

const EnvReattachProviders = "TF_REATTACH_PROVIDERS"

type BitwardenHolder interface {
	Bitwarden() (*bitwarden.Bitwarden, error)
}

func NewPlugin() *Plugin {
	return &Plugin{
		ProviderAddr: "registry.terraform.io/echocat/bitwarden",
	}
}

type Plugin struct {
	ProviderAddr string

	cancelFunc context.CancelFunc
	closeCh    <-chan struct{}
	config     plugin.ReattachConfig

	BitwardenHolder BitwardenHolder
}

func (this *Plugin) RegisterFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("terraform.plugin.address", "").
		Default(this.ProviderAddr).
		StringVar(&this.ProviderAddr)
}

func (this *Plugin) GetReattachProviders() map[string]plugin.ReattachConfig {
	result := map[string]plugin.ReattachConfig{}

	plain := strings.TrimSpace(os.Getenv(EnvReattachProviders))
	if plain != "" {
		if err := json.Unmarshal([]byte(plain), &result); err != nil {
			log.WithError(err).
				Warnf("%s contains illegal configuration. Skipping it.", EnvReattachProviders)
		}
	}

	return result
}

func (this *Plugin) GetEnrichedReattachProviders() string {
	providers := this.GetReattachProviders()

	providers[this.ProviderAddr] = this.config

	b, err := json.Marshal(providers)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func (this *Plugin) Initialize() error {
	ctx, cancel := context.WithCancel(context.Background())
	config, closeCh, err := plugin.DebugServe(ctx, this.toOpts())
	if err != nil {
		cancel()
		return err
	}
	this.closeCh = closeCh
	this.cancelFunc = cancel
	this.config = config
	return nil
}

func (this *Plugin) Close() error {
	defer func() {
		this.config = plugin.ReattachConfig{}
	}()

	defer func() {
		this.closeCh = nil
	}()
	defer func() {
		if v := this.closeCh; v != nil {
			<-v
		}
	}()

	defer func() {
		this.cancelFunc = nil
	}()
	defer func() {
		if v := this.cancelFunc; v != nil {
			v()
		}
	}()

	return nil
}

func (this *Plugin) ShouldServe() bool {
	_, tfPluginCookie := os.LookupEnv(plugin.Handshake.MagicCookieKey)
	return tfPluginCookie || this.isDebug()
}

func (this *Plugin) isDebug() bool {
	_, v := os.LookupEnv("TF_PLUGIN_DEBUG")
	return v
}

func (this *Plugin) Serve() {
	if this.isDebug() {
		if err := plugin.Debug(context.Background(), this.ProviderAddr, this.toOpts()); err != nil {
			panic(err)
		}
	} else {
		plugin.Serve(this.toOpts())
	}
}

func (this *Plugin) toOpts() *plugin.ServeOpts {
	logger := &Logger{log.GetRootLogger()}
	hclog.SetDefault(logger)
	_ = os.Setenv("TF_LOG_SDK_PROTO", "error")
	_ = os.Setenv("TF_LOG_SDK", "error")
	return &plugin.ServeOpts{
		ProviderFunc:        this.provider,
		Logger:              logger,
		NoLogOutputOverride: true,
	}
}
