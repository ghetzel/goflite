package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/goflite"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger(`main`)

func main() {
	app := cli.NewApp()
	app.Name = `say`
	app.Usage = `A voice synthesis utility/`
	app.ArgsUsage = `MESSAGE`
	app.Version = `0.0.14`
	app.EnableBashCompletion = false

	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `voice, V`,
			Usage: `Specifies the voice to synthesize.`,
		},
		cli.DurationFlag{
			Name:  `post-finish-delay, d`,
			Usage: `Specifies how long to wait after the buffers drain before exiting.`,
			Value: goflite.DefaultPostFinishDelay,
		},
		cli.IntFlag{
			Name:  `target-mean, M`,
			Usage: `Affects the pitch of the voice`,
			Value: 160,
		},
		cli.IntFlag{
			Name:  `target-stddev, D`,
			Usage: `Affects the vibrato of the voice`,
			Value: 25,
		},
		cli.Float64Flag{
			Name:  `stretch, S`,
			Usage: `Applies a factor to speed up or slow down the voice`,
			Value: 1.0,
		},
	}

	app.Before = func(c *cli.Context) error {
		var addlInfo string
		levels := append([]string{
			`info`,
		}, c.StringSlice(`log-level`)...)

		for _, levelspec := range levels {
			var levelName string
			var moduleName string

			if parts := strings.SplitN(levelspec, `:`, 2); len(parts) == 1 {
				levelName = parts[0]
			} else {
				moduleName = parts[0]
				levelName = parts[1]
			}

			if level, err := logging.LogLevel(levelName); err == nil {
				if level == logging.DEBUG {
					addlInfo = `%{module}: `
				}

				logging.SetLevel(level, moduleName)
			} else {
				return err
			}
		}

		logging.SetFormatter(logging.MustStringFormatter(
			fmt.Sprintf("%%{color}%%{level:.4s}%%{color:reset}[%%{id:04d}] %s%%{message}", addlInfo),
		))

		log.Debugf("Starting %s %s", c.App.Name, c.App.Version)
		return nil
	}

	app.Action = func(c *cli.Context) {
		if entries, err := ioutil.ReadDir(`.`); err == nil {
			for _, entry := range entries {
				if ext := path.Ext(entry.Name()); entry.Mode().IsRegular() && ext == `.flitevox` {
					voiceName := strings.TrimSuffix(path.Base(entry.Name()), ext)

					if v := c.String(`voice`); v != `` && v != voiceName {
						continue
					}

					if err := goflite.AddVoice(
						voiceName,
						entry.Name(),
					); err == nil {
						log.Debugf("added voice %v (%v)", voiceName, entry.Name())
					} else {
						log.Warningf("failed to add voice %v: %v", entry.Name(), err)
					}
				}
			}
		}

		if synth, err := goflite.NewSynthesizer(); err == nil {
			if err := synth.SetVoice(c.String(`voice`)); err != nil {
				log.Fatalf("failed to set voice: %v", err)
				return
			}

			synth.PostFinishDelay = c.Duration(`post-finish-delay`)
			synth.SetIntFeature(`int_f0_target_mean`, int64(c.Int(`target-mean`)))
			synth.SetIntFeature(`int_f0_target_stddev`, int64(c.Int(`target-stddev`)))
			synth.SetFloatFeature(`duration_stretch`, c.Float64(`stretch`))

			input := c.Args().First()

			if input == `` {
				if data, err := ioutil.ReadAll(os.Stdin); err == nil {
					if len(data) > 0 {
						input = string(data)
					}
				}
			}

			if err := synth.Say(input); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatalf("failed to create synthesizer: %v", err)
		}
	}

	app.Run(os.Args)
}
