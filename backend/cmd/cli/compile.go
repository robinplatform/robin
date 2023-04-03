package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"

	"robinplatform.dev/internal/compile/compileClient"
	"robinplatform.dev/internal/compile/toolkit"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/project"
)

type CompileCommand struct {
	appId              string
	enablePprof        bool
	enableTrace        bool
	forceStableToolkit bool
}

func (cmd *CompileCommand) Name() string {
	return "compile"
}

func (cmd *CompileCommand) Description() string {
	return "compile a robin app"
}

func (cmd *CompileCommand) Parse(flagSet *flag.FlagSet, args []string) error {
	perfDefaultOn := config.GetReleaseChannel() == config.ReleaseChannelDev
	flagSet.BoolVar(&cmd.enablePprof, "pprof", perfDefaultOn, "Enable profiling using runtime/pprof")
	flagSet.BoolVar(&cmd.enableTrace, "trace", perfDefaultOn, "Enable tracing using runtime/trace")
	flagSet.StringVar(&cmd.appId, "appId", "", "App ID")

	if config.GetReleaseChannel() != config.ReleaseChannelStable {
		flagSet.BoolVar(&cmd.forceStableToolkit, "use-stable-toolkit", false, "Force the use of the stable toolkit")
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	return nil
}

func (cmd *CompileCommand) Run() error {
	_, err := project.LoadFromEnv()
	if err != nil {
		return err
	}

	if cmd.forceStableToolkit {
		toolkit.DisableEmbeddedToolkit()
	}

	if cmd.enablePprof {
		out, err := os.Create("./profile.pprof")
		if err != nil {
			return fmt.Errorf("failed to make profile file")
		}

		pprof.StartCPUProfile(out)
		defer pprof.StopCPUProfile()
	}

	if cmd.enableTrace {
		out, err := os.Create("./trace.trace")
		if err != nil {
			return fmt.Errorf("failed to make trace file")
		}

		trace.Start(out)
		defer trace.Stop()
	}

	client, err := httpcache.NewClient(config.GetHttpCachePath(), 1024*1024*128)
	if err != nil {
		return err
	}

	_, err = compileClient.BuildClientBundle(compileClient.ClientJSInput{
		AppId:           cmd.appId,
		HttpClient:      client,
		DefineConstants: make(map[string]string),
	})
	if err != nil {
		return err
	}

	err = client.Save()

	return err
}
