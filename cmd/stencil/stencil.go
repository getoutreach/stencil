package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	oapp "github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/gobox/pkg/cfg"
	olog "github.com/getoutreach/gobox/pkg/log"
	"github.com/getoutreach/gobox/pkg/secrets"
	"github.com/getoutreach/gobox/pkg/trace"
	"github.com/getoutreach/gobox/pkg/updater"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	// Place any extra imports for your startup code here
	///Block(imports)

	"github.com/getoutreach/stencil/internal/stencil"
	"github.com/getoutreach/stencil/pkg/codegen"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/processors"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	///EndBlock(imports)
)

// Why: We can't compile in things as a const.
//nolint:gochecknoglobals
var (
	HoneycombTracingKey = "NOTSET"
)

///Block(global)
///EndBlock(global)

func overrideConfigLoaders() {
	var fallbackSecretLookup func(context.Context, string) ([]byte, error)
	fallbackSecretLookup = secrets.SetDevLookup(func(ctx context.Context, key string) ([]byte, error) {
		if key == "APIKey" {
			return []byte(HoneycombTracingKey), nil
		}

		return fallbackSecretLookup(ctx, key)
	})

	olog.SetOutput(ioutil.Discard)

	fallbackConfigReader := cfg.DefaultReader()
	cfg.SetDefaultReader(func(fileName string) ([]byte, error) {
		if fileName == "trace.yaml" {
			traceConfig := &trace.Config{
				Honeycomb: trace.Honeycomb{
					Enabled: true,
					APIHost: "https://api.honeycomb.io",
					APIKey: cfg.Secret{
						Path: "APIKey",
					},
					///Block(dataset)
					Dataset: "dev-tooling-team",
					///EndBlock(dataset)
					SamplePercent: 100,
				},
			}
			b, err := yaml.Marshal(&traceConfig)
			if err != nil {
				panic(err)
			}
			return b, nil
		}

		return fallbackConfigReader(fileName)
	})
}

func main() { //nolint:funlen,gocyclo
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	log := logrus.New()

	exitCode := 0
	cli.OsExiter = func(code int) { exitCode = code }

	oapp.SetName("stencil")
	overrideConfigLoaders()

	// handle ^C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		out := <-c
		log.Debugf("shutting down: %v", out)
		cancel()
	}()

	if err := trace.InitTracer(ctx, "stencil"); err != nil {
		log.WithError(err).Debugf("failed to start tracer")
	}
	ctx = trace.StartTrace(ctx, "stencil")

	///Block(init)
	///EndBlock(init)

	exit := func() {
		trace.End(ctx)
		trace.CloseTracer(ctx)
		///Block(exit)
		///EndBlock(exit)
		os.Exit(exitCode)
	}
	defer exit()

	// wrap everything around a call as this ensures any panics
	// are caught and recorded properly
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic %v", r)
		}
	}()
	ctx = trace.StartCall(ctx, "main")
	defer trace.EndCall(ctx)

	app := cli.App{
		Version: oapp.Version,
		Name:    "stencil",
		///Block(app)
		Action: func(c *cli.Context) error {
			log.Infof("stencil %s", oapp.Version)

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to get the current working directory")
			}

			serviceManifest, err := configuration.NewDefaultServiceManifest()
			if err != nil {
				return errors.Wrap(err, "failed to parse service.yaml")
			}

			if !stencil.ValidateName(serviceManifest.Name) {
				return fmt.Errorf("'%s' is not an acceptable package name", serviceManifest.Name)
			}

			_, err = git.PlainOpen(cwd)
			if err != nil {
				log.Info("creating git repository")
				_, err = git.PlainInit(cwd, false)
				if err != nil {
					return errors.Wrap(err, "failed to initialize git repository")
				}
			}

			b := codegen.NewBuilder(filepath.Base(cwd), cwd, serviceManifest,
				c.String("github-ssh-key"), cfg.SecretData(c.String("github-access-token")))

			warnings, err := b.Run(ctx, log)
			for _, warning := range warnings {
				log.Warn(warning)
			}

			if err != nil {
				return errors.Wrap(err, "run codegen")
			}

			log.Info("Running post-processors")
			err = filepath.Walk(".", func(fp string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.Mode()&os.ModeSymlink == os.ModeSymlink {
					// Skip symlinks.
					return nil
				}

				f, err := os.Open(fp)
				if err != nil {
					return errors.Wrapf(err, "open file '%s'", fp)
				}
				defer f.Close()

				processedFile, err := b.Processors.Process(true, processors.NewFile(f, fp), nil)
				if err != nil && err != processors.ErrNotProcessable {
					return errors.Wrap(err, "failed to process file")
				} else if err == processors.ErrNotProcessable {
					// Skip file.
					return nil
				}

				perms := os.FileMode(0644)
				if strings.HasSuffix(fp, ".sh") {
					perms = os.FileMode(0744)
				}

				data, err := ioutil.ReadAll(processedFile)
				if err != nil {
					return errors.Wrap(err, "read processed file")
				}

				log.Infof("Processed file '%s' in post-processing step", fp)
				if err := ioutil.WriteFile(fp, data, perms); err != nil {
					return errors.Wrap(err, "failed to write post-processed file")
				}

				return nil
			})

			if err != nil {
				return errors.Wrap(err, "run post-processors on all files")
			}

			return b.FormatFiles(ctx)
		},
		///EndBlock(app)
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "skip-update",
			Usage: "skips the updater check",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "enables debug logging for all components (i.e updater)",
		},
		&cli.BoolFlag{
			Name:  "enable-prereleases",
			Usage: "Enable considering pre-releases when checking for updates",
		},
		&cli.BoolFlag{
			Name:  "force-update-check",
			Usage: "Force checking for an update",
		},
		///Block(flags)
		&cli.BoolFlag{
			Name:  "dev",
			Usage: "Use local manifests instead of remote ones, useful for development",
		},
		&cli.StringFlag{
			Name:    "github-ssh-key",
			Usage:   "SSH Key to use to download templates with, if not set ~/.ssh/config is read and falls back to ssh-agent",
			EnvVars: []string{"GITHUB_SSH_KEY"},
		},
		&cli.StringFlag{
			Name:    "github-access-token",
			Usage:   "Github Access Token (or Personal Access Token) to use for downloading templates",
			EnvVars: []string{"GITHUB_ACCESS_TOKEN"},
		},
		///EndBlock(flags)
	}
	app.Commands = []*cli.Command{
		///Block(commands)
		///EndBlock(commands)
	}

	app.Before = func(c *cli.Context) error {
		///Block(before)
		_, err := box.EnsureBox(ctx, []string{
			"git@github.com:getoutreach/box",
		}, log)
		if err != nil {
			return err
		}
		///EndBlock(before)

		// add info to the root trace about our command and args
		cargs := c.Args().Slice()
		command := ""
		args := make([]string, 0)
		if len(cargs) > 0 {
			command = c.Args().Slice()[0]
		}
		if len(cargs) > 1 {
			args = cargs[1:]
		}

		userName := ""
		if u, err := user.Current(); err == nil {
			userName = u.Username
		}
		trace.AddInfo(ctx, olog.F{
			"stencil.subcommand": command,
			"stencil.args":       strings.Join(args, " "),
			"os.user":            userName,
			"os.name":            runtime.GOOS,
			///Block(trace)
			///EndBlock(trace)
		})

		// restart when updated
		traceCtx := trace.StartCall(c.Context, "updater.NeedsUpdate") //nolint:govet
		defer trace.EndCall(traceCtx)

		// restart when updated
		if updater.NeedsUpdate(traceCtx, log, "", oapp.Version, c.Bool("skip-update"), c.Bool("debug"), c.Bool("enable-prereleases"), c.Bool("force-update-check")) {
			log.Infof("stencil has been updated, please re-run your command")
			exitCode = 5
			trace.EndCall(traceCtx)
			exit()
		}

		return nil
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Errorf("failed to run: %v", err)
		//nolint:errcheck // We're attaching the error to the trace.
		trace.SetCallStatus(ctx, err)
		exitCode = 1

		return
	}
}
