package backend

import (
	"context"
	"fmt"
	backend "github.com/bhoriuchi/terraform-backend-http/go"
	"github.com/echocat/slf4g"
	"github.com/echocat/terraform-provider-bitwarden/bitwarden"
	"github.com/echocat/terraform-provider-bitwarden/plugin"
	"github.com/echocat/terraform-provider-bitwarden/utils"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func NewBackend(p *plugin.Plugin) *Backend {
	result := Backend{
		config:        NewConfig(),
		overlayConfig: NewConfig(),

		plugin: p,
	}
	p.BitwardenHolder = &result
	uv := true
	result.config.Bitwarden.UnlockIfRequired = &uv
	return &result
}

type Backend struct {
	TerraformArgs []string

	ExitCode int

	config        *Config
	overlayConfig *Config
	plugin        *plugin.Plugin
	bitwarden     *bitwarden.Bitwarden
	backend       *backend.Backend
	server        *http.Server
	listener      net.Listener
}

func (this *Backend) RegisterFlags(app *kingpin.Application) {
	app.Flag("item.id", "Item name which holds the state of this terraform environment.").
		Envar("TF_BACKEND_ITEM_NAME").
		StringVar(&this.overlayConfig.State.ItemName)
	app.Flag("item.name", "Item ID which holds the state of this terraform environment.").
		Envar("TF_BACKEND_ITEM_ID").
		StringVar(&this.overlayConfig.State.ItemId)
	app.Flag("listen", "Port where the backend is listening while the execution to (at localhost).").
		Envar("TF_BACKEND_PORT").
		Uint16Var(&this.overlayConfig.Backend.Port)
	app.Flag("terraform.executable", "Executable of Terraform to execute (if required).").
		Envar("TF_EXECUTABLE").
		SetValue(&this.overlayConfig.Terraform.Executable)
	app.Flag("bitwarden.executable", "Executable of Bitwarden CLI to use.").
		Envar("BW_CLI_EXECUTABLE").
		StringVar(&this.overlayConfig.Bitwarden.Executable)
	app.Flag("bitwarden.session", "Existing Bitwarden session to use.").
		Envar("BW_SESSION").
		StringVar(&this.overlayConfig.Bitwarden.Session)
	app.Flag("bitwarden.unlock", "Will unlock Bitwraden (if required, enabled by default).").
		Envar("BW_UNLOCK").
		BoolVar(this.overlayConfig.Bitwarden.UnlockIfRequired)

	cmd := app.Command("wrap", "Starts the backend and calls terraform accordingly.").
		Default().
		Action(this.cmdExecute)
	cmd.Arg("args", "Arguments to pass to terraform.").
		Required().
		StringsVar(&this.TerraformArgs)

	this.plugin.RegisterFlags(cmd)
}

func (this *Backend) cmdExecute(*kingpin.ParseContext) (rErr error) {
	if err := this.Initialize(); err != nil {
		return err
	}
	defer func() {
		if err := this.Close(); err != nil && rErr == nil {
			rErr = err
		}
	}()

	return this.runTerraform()
}

func (this *Backend) runTerraform() error {
	executable := this.config.GetTerraform().GetExecutable()
	cmd, err := executable.Command(this.TerraformArgs...)
	if err != nil {
		return executable.Errorf(this.TerraformArgs, "%w", err)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env, err = this.terraformEnvironment()
	if err != nil {
		return executable.Errorf(this.TerraformArgs, "%w", err)
	}

	err = cmd.Run()
	if eErr, ok := err.(*exec.ExitError); ok {
		this.ExitCode = eErr.ExitCode()
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func (this *Backend) Initialize() error {
	if err := this.config.Read(nil); err != nil {
		return err
	}
	nc := this.overlayConfig.Merge(*this.config)

	if err := nc.Validate(); err != nil {
		return err
	}

	this.config = &nc

	var success bool
	if err := this.plugin.Initialize(); err != nil {
		return err
	}
	defer func() {
		if !success {
			_ = this.plugin.Close()
		}
	}()

	b, err := this.newBitwarden()
	if err != nil {
		return err
	}

	this.backend = backend.NewBackend(&Store{this}, &backend.Options{
		Logger:          this.logHook,
		GetMetadataFunc: this.getMetaDataHook,
		GetRefFunc:      this.getRefHook,
	})
	if err := this.backend.Init(); err != nil {
		return fmt.Errorf("cannot initialze terraform backend: %w", err)
	}

	s, ln, err := this.newServer()
	if err != nil {
		return err
	}
	defer func() {
		if !success {
			_ = s.Close()
		}
	}()

	this.bitwarden = b
	this.listener = ln
	this.server = s

	success = true
	return nil
}

func (this *Backend) newBitwarden() (*bitwarden.Bitwarden, error) {
	bc := this.config.GetBitwarden()
	b, err := bc.NewBitwarden()
	if err != nil {
		return nil, err
	}
	if bc.IsUnlockIfRequired() {
		if err := b.Unlock(true); err != nil {
			return nil, err
		}
	} else {
		if ok, err := b.Test(); err != nil {
			return nil, err
		} else if !ok {
			return nil, bitwarden.ErrWrongSession
		}
	}

	return b, err
}

func (this *Backend) logHook(level, message string, err error) {
	l := log.GetRootLogger()
	if err != nil {
		l = l.WithError(err)
	}
	switch level {
	case "debug":
		l.Debug(message)
	case "error":
		l.Error(message)
	default:
		l.Info(message)
	}
}

func (this *Backend) getRefHook(r *http.Request) string {
	return r.URL.Query().Get("item")
}

func (this *Backend) getMetaDataHook(map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{}
}

func (this *Backend) newServer() (*http.Server, net.Listener, error) {
	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", this.config.GetBackend().GetPort()),
		Handler: http.HandlerFunc(this.handle),
	}
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.WithError(err).
				With("address", this.server.Addr).
				Error("Failed to listen.")
			os.Exit(67)
		}
	}()

	return server, ln, err
}

func (this *Backend) handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "LOCK":
		this.backend.HandleLockState(w, r)
	case "UNLOCK":
		this.backend.HandleUnlockState(w, r)
	case http.MethodGet:
		this.backend.HandleGetState(w, r)
	case http.MethodPost:
		this.backend.HandleUpdateState(w, r)
	case http.MethodDelete:
		this.backend.HandleDeleteState(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (this *Backend) Close() (rErr error) {
	defer func() {
		this.bitwarden = nil
	}()

	defer func() {
		this.listener = nil
	}()
	defer func() {
		if v := this.listener; v != nil {
			if err := v.Close(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") && rErr == nil {
				rErr = err
			}
		}
	}()

	defer func() {
		this.server = nil
	}()
	defer func() {
		if v := this.server; v != nil {
			if err := v.Shutdown(context.Background()); err != nil && !strings.Contains(err.Error(), "use of closed network connection") && rErr == nil {
				rErr = err
			}
		}
	}()

	return
}

func (this *Backend) Bitwarden() (*bitwarden.Bitwarden, error) {
	if v := this.bitwarden; v != nil {
		return v, nil
	}
	return nil, fmt.Errorf("backend not yet initialized")
}

func (this *Backend) Plugin() *plugin.Plugin {
	return this.plugin
}

func (this *Backend) baseAddress() (string, error) {
	sc := this.config.GetState()
	bc := this.config.GetBackend()
	if sc.ItemId != "" {
		return fmt.Sprintf("http://127.0.0.1:%d/?item=id:%v", bc.GetPort(), url.QueryEscape(sc.ItemId)), nil
	} else if sc.ItemName != "" {
		return fmt.Sprintf("http://127.0.0.1:%d/?item=name:%v", bc.GetPort(), url.QueryEscape(sc.ItemName)), nil
	} else {
		return "", fmt.Errorf("config.state has to contain either item_id or item_name")
	}
}

func (this *Backend) terraformEnvironment() ([]string, error) {
	b, err := this.Bitwarden()
	if err != nil {
		return nil, err
	}
	baseAddress, err := this.baseAddress()
	if err != nil {
		return nil, err
	}

	env := map[string]string{
		"BW_SESSION": b.Session(),

		plugin.EnvReattachProviders: this.Plugin().GetEnrichedReattachProviders(),

		"TF_BACKEND":             "http",
		"TF_HTTP_ADDRESS":        baseAddress,
		"TF_HTTP_LOCK_ADDRESS":   baseAddress,
		"TF_HTTP_UNLOCK_ADDRESS": baseAddress,
		"TF_HTTP_RETRY_MAX":      "0",
	}

	vars, err := this.config.Variables.Resolve(b)
	if err != nil {
		return nil, err
	}
	for k, v := range vars {
		env["TF_VAR_"+k] = v
	}

	return utils.AddEnvironment(os.Environ(), env), nil
}

func (this *Backend) GetOrganizationId() string {
	return this.config.GetState().OrganizationId
}

func (this *Backend) GetCollectionId() string {
	return this.config.GetState().CollectionId
}

func (this *Backend) GetFolderId() string {
	return this.config.GetState().FolderId
}

func (this *Backend) GetConfig() Config {
	return *this.config
}
