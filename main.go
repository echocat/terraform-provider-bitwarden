package main

import (
	"fmt"
	"github.com/echocat/slf4g/native"
	"github.com/echocat/slf4g/native/facade/value"
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

	lv := value.NewProvider(native.DefaultProvider)
	app.Flag("log.level", "Level how messages should be logged.").
		Default(lv.Level.String()).
		Envar("TF_BACKEND_LOG_LEVEL").
		SetValue(lv.Level)
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
