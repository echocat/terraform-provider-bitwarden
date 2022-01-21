package main

import (
	"bytes"
	"fmt"
	log "github.com/echocat/slf4g"
	"github.com/echocat/slf4g/level"
	"github.com/echocat/slf4g/native"
	"github.com/echocat/slf4g/native/facade/value"
	sdk "github.com/echocat/slf4g/sdk/bridge"
	"github.com/echocat/terraform-provider-bitwarden/backend"
	"github.com/echocat/terraform-provider-bitwarden/plugin"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

func main() {
	p := plugin.NewPlugin()
	if p.ShouldServe() {
		p.Serve()
		return
	}

	app := kingpin.New(os.Args[0], "Provides a Bitwarden based backend for terraform.")

	var chdir, oldDir string
	configureLog(app)
	app.Flag("chdir", "Directory to move into while executing the actions.").
		Envar("TF_CHROOT").
		StringVar(&chdir)

	app.PreAction(func(*kingpin.ParseContext) error {
		if chdir != "" {
			path, err := os.Getwd()
			if err != nil {
				return err
			}
			oldDir = path
			if err := os.Chdir(chdir); err != nil {
				return fmt.Errorf("cannot chdir: %w", err)
			}
		}
		return nil
	})
	defer func() {
		if oldDir != "" {
			_ = os.Chdir(oldDir)
		}
	}()

	b := backend.NewBackend(p)
	defer func() {
		_ = b.Close()
	}()
	b.RegisterFlags(app)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		if b.ExitCode > 0 {
			os.Exit(b.ExitCode)
		} else {
			os.Exit(69)
		}
	}

	os.Exit(b.ExitCode)

}

func configureLog(app *kingpin.Application) {
	sdk.Configure(func(v *log.LoggingWriter) {
		v.Interceptor = interceptSdkLog
	})

	lv := value.NewProvider(native.DefaultProvider)
	app.Flag("log.level", "Level how messages should be logged.").
		Default(lv.Level.String()).
		Envar("TF_BACKEND_LOG_LEVEL").
		SetValue(lv.Level)
}

var (
	warnPrefix1  = []byte("[WARN] ")
	warnPrefix2  = []byte("WARN: ")
	errorPrefix1 = []byte("[ERROR] ")
	errorPrefix2 = []byte("ERROR: ")
)

func interceptSdkLog(b []byte, lvl level.Level) ([]byte, level.Level, error) {
	if bytes.HasPrefix(b, warnPrefix1) {
		return b[len(warnPrefix1):], level.Warn, nil
	}
	if bytes.HasPrefix(b, warnPrefix2) {
		return b[len(warnPrefix2):], level.Warn, nil
	}
	if bytes.HasPrefix(b, errorPrefix1) {
		return b[len(errorPrefix1):], level.Error, nil
	}
	if bytes.HasPrefix(b, errorPrefix2) {
		return b[len(errorPrefix2):], level.Error, nil
	}
	return b, lvl, nil
}
