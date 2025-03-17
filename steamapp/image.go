package steamapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
	xio "github.com/frantjc/x/io"
	xslice "github.com/frantjc/x/slice"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/util/progress/progressui"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
)

type BuildImageOpts struct {
	BaseImageRef string

	AptPkgs []string

	SteamcmdImageRef   string
	Beta, BetaPassword string
	LaunchType         string
	PlatformType       steamcmd.PlatformType
	Dir                string

	Execs []string

	User       string
	Entrypoint []string
	Cmd        []string

	Output     io.WriteCloser
	ExportType string
}

func (o *BuildImageOpts) Apply(opts *BuildImageOpts) {
	if o == nil {
		return
	}
	if o.BaseImageRef != "" {
		opts.BaseImageRef = o.BaseImageRef
	}
	if len(o.AptPkgs) > 0 {
		opts.AptPkgs = o.AptPkgs
	}
	if o.SteamcmdImageRef != "" {
		opts.SteamcmdImageRef = o.SteamcmdImageRef
	}
	if len(o.Execs) > 0 {
		opts.Execs = o.Execs
	}
	if o.Beta != "" {
		opts.Beta = o.Beta
	}
	if o.BetaPassword != "" {
		opts.BetaPassword = o.BetaPassword
	}
	if o.PlatformType.String() != "" {
		opts.PlatformType = o.PlatformType
	}
	if o.LaunchType != "" {
		opts.LaunchType = o.LaunchType
	}
	if o.Dir != "" {
		opts.Dir = o.Dir
	}
	if o.User != "" {
		opts.User = o.User
	}
	if len(o.Entrypoint) > 0 {
		opts.Entrypoint = o.Entrypoint
	}
	if len(o.Cmd) > 0 {
		opts.Cmd = o.Cmd
	}
	if o.Output != nil {
		opts.Output = o.Output
	}
	if o.ExportType != "" {
		opts.ExportType = o.ExportType
	}
}

type BuildImageOpt interface {
	Apply(*BuildImageOpts)
}

type ImageBuilder struct {
	*client.Client
}

const (
	DefaultUser             = "steam"
	DefaultDir              = "/home/" + DefaultUser
	DefaultLaunchType       = "server"
	DefaultBaseImageRef     = "docker.io/library/debian:stable-slim"
	DefaultSteamcmdImageRef = "docker.io/steamcmd/steamcmd:latest"
	DefaultExportType       = client.ExporterDocker
)

func getImageConfig(ctx context.Context, appID int, opts *BuildImageOpts) (*specs.ImageConfig, int, error) {
	ref, err := name.ParseReference(opts.BaseImageRef)
	if err != nil {
		return nil, 0, err
	}

	img, err := remote.Image(ref, remote.WithContext(ctx))
	if err != nil {
		return nil, 0, err
	}

	cfgf, err := img.ConfigFile()
	if err != nil {
		return nil, 0, err
	}

	appInfo, err := appinfoutil.GetAppInfo(ctx, appID)
	if err != nil {
		return nil, 0, err
	}

	icfg := &specs.ImageConfig{
		User:         cfgf.Config.User,
		ExposedPorts: cfgf.Config.ExposedPorts,
		Env:          cfgf.Config.Env,
		Entrypoint:   opts.Entrypoint,
		Cmd:          opts.Cmd,
		Volumes:      cfgf.Config.Volumes,
		WorkingDir:   opts.Dir,
		Labels:       cfgf.Config.Labels,
		StopSignal:   cfgf.Config.StopSignal,
		ArgsEscaped:  cfgf.Config.ArgsEscaped,
	}

	if opts.User != "" {
		icfg.User = opts.User
	}

	for _, launch := range appInfo.Config.Launch {
		if launch.Config != nil && strings.Contains(launch.Config.OSList, opts.PlatformType.String()) {
			if opts.LaunchType == "" || strings.EqualFold(launch.Type, opts.LaunchType) {
				if icfg.Labels == nil {
					icfg.Labels = map[string]string{}
				}
				icfg.Labels["cc.frantj.sindri.id"] = fmt.Sprint(appID)
				icfg.Labels["cc.frantj.sindri.name"] = appInfo.Common.Name
				icfg.Labels["cc.frantj.sindri.type"] = appInfo.Common.Type
				branchName := DefaultBranchName
				if opts.Beta != "" {
					branchName = opts.Beta
				}
				var buildID int
				icfg.Labels["cc.frantj.sindri.branch"] = branchName
				if branch, ok := appInfo.Depots.Branches[branchName]; ok {
					buildID = branch.BuildID
					icfg.Labels["cc.frantj.sindri.buildid"] = fmt.Sprint(branch.BuildID)
					icfg.Labels["cc.frantj.sindri.description"] = branch.Description
				}
				if icfg.Entrypoint == nil {
					icfg.Entrypoint = []string{filepath.Join(opts.Dir, launch.Executable)}
				}
				if icfg.Cmd == nil {
					icfg.Cmd = xslice.Filter(regexp.MustCompile(`\s+`).Split(launch.Arguments, -1), func(arg string, _ int) bool {
						return arg != ""
					})
					if len(icfg.Cmd) == 0 {
						icfg.Cmd = nil
					}
				}
				return icfg, buildID, nil
			}
		}
	}

	return nil, 0, fmt.Errorf("app ID %d does not support %s, only %s", appInfo.Common.GameID, opts.PlatformType, appInfo.Common.OSList)
}

func getDefinition(ctx context.Context, appID, buildID int, opts *BuildImageOpts) (*llb.Definition, error) {
	arg, err := steamcmd.Args(nil,
		steamcmd.ForceInstallDir(filepath.Join("/mnt", opts.Dir)),
		steamcmd.Login{},
		steamcmd.ForcePlatformType(opts.PlatformType),
		steamcmd.AppUpdate{
			AppID:        appID,
			Beta:         opts.Beta,
			BetaPassword: opts.BetaPassword,
		},
		steamcmd.Quit,
	)
	if err != nil {
		return nil, err
	}

	state := llb.Image(opts.BaseImageRef)

	if len(opts.AptPkgs) > 0 {
		state = state.
			Run(llb.Shlex("apt-get update -y")).
			Run(llb.Shlexf("apt-get install -y --no-install-recommends %s", strings.Join(opts.AptPkgs, " "))).
			Run(llb.Shlex("rm -rf /var/lib/apt/lists/*")).
			Run(llb.Shlex("apt-get clean")).
			Root()
	}

	steamcmdState := llb.Image(opts.SteamcmdImageRef)

	if opts.User != "" {
		// This creates /home/steam, which the `steamcmd app_update` command
		// below needs to exist when using [DefaultDir].
		state = state.
			Run(llb.Shlexf("groupadd --system %s", opts.User)).
			Run(llb.Shlexf("useradd --system --gid %s --shell /bin/bash --create-home %s", opts.User, opts.User)).
			Root()

		steamcmdState = steamcmdState.
			Run(llb.Shlexf("groupadd --system %s", opts.User)).
			Run(llb.Shlexf("useradd --system --gid %s --shell /bin/bash --create-home %s", opts.User, opts.User)).
			User(opts.User)
	}

	state = steamcmdState.
		// `echo`ing the buildid here is to workaround
		// buildkit cacheing the steamcmd app_update command
		// when there has been a new build pushed to the branch.
		Run(llb.Shlexf("echo %d", buildID)).
		Run(llb.Shlexf("steamcmd %s", strings.Join(arg, " "))).
		AddMount("/mnt", state).
		Dir(opts.Dir)

	for _, exec := range opts.Execs {
		state = state.
			Run(llb.Shlex(exec)).
			Root()
	}

	if opts.User != "" {
		state = state.User(opts.User)
	}

	return state.Marshal(ctx, llb.LinuxAmd64)
}

func getSolveOpt(ctx context.Context, appID int, opts *BuildImageOpts) (*client.SolveOpt, int, error) {
	icfg, buildID, err := getImageConfig(ctx, appID, opts)
	if err != nil {
		return nil, 0, err
	}

	ib, err := json.Marshal(&specs.Image{
		Config: *icfg,
		Platform: specs.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
	})
	if err != nil {
		return nil, 0, err
	}

	return &client.SolveOpt{
		Exports: []client.ExportEntry{
			{
				Type: opts.ExportType,
				Attrs: map[string]string{
					exptypes.ExporterImageConfigKey: string(ib),
				},
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return opts.Output, nil
				},
			},
		},
	}, buildID, nil
}

func (a *ImageBuilder) BuildImage(ctx context.Context, appID int, opts ...BuildImageOpt) error {
	o := &BuildImageOpts{
		BaseImageRef:     DefaultBaseImageRef,
		SteamcmdImageRef: DefaultSteamcmdImageRef,
		Output: xio.WriterCloser{
			Writer: io.Discard,
			Closer: xio.CloserFunc(func() error {
				return nil
			}),
		},
		Dir:          DefaultDir,
		LaunchType:   DefaultLaunchType,
		PlatformType: steamcmd.PlatformTypeLinux,
		User:         DefaultUser,
		ExportType:   DefaultExportType,
	}

	for _, opt := range opts {
		opt.Apply(o)
	}

	solvOpt, buildID, err := getSolveOpt(ctx, appID, o)
	if err != nil {
		return err
	}

	def, err := getDefinition(ctx, appID, buildID, o)
	if err != nil {
		return err
	}

	solvStatusC := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		if _, err = a.Solve(ctx, def, *solvOpt, solvStatusC); err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		d, err := progressui.NewDisplay(io.Discard, progressui.AutoMode)
		if err != nil {
			return err
		}

		if _, err = d.UpdateFrom(ctx, solvStatusC); err != nil {
			return err
		}

		return nil
	})

	return eg.Wait()
}
