package steamapp

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/util/progress/progressui"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
					icfg.Cmd = slices.DeleteFunc(regexp.MustCompile(`\s+`).Split(launch.Arguments, -1), func(arg string) bool {
						return arg == ""
					})
					if len(icfg.Cmd) == 0 {
						icfg.Cmd = nil
					}
				}
				return icfg, buildID, nil
			}
		}
	}

	return nil, 0, fmt.Errorf(
		"app ID %d launch type %s does not support %s",
		appInfo.Common.GameID, opts.LaunchType, opts.PlatformType,
	)
}

func GetImageConfig(ctx context.Context, appID int, opts ...BuildImageOpt) (*specs.ImageConfig, error) {
	imageConfig, _, err := getImageConfig(ctx, appID, newBuildImageOpts(opts...))
	return imageConfig, err
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
			Run(shlexf("apt-get update -y && apt-get install -y --no-install-recommends %s && rm -rf /var/lib/apt/lists/* && apt-get clean", strings.Join(opts.AptPkgs, " "))).
			Root()
	}

	steamcmdState := llb.Image(opts.SteamcmdImageRef)

	if opts.User != "" {
		// This creates /home/steam, which the `steamcmd app_update` command
		// below needs to exist when using [DefaultDir].
		state = state.
			Run(shlexf("groupadd --system %s && useradd --system --gid %s --shell /bin/bash --create-home %s", opts.User, opts.User, opts.User)).
			Root()

		steamcmdState = steamcmdState.
			Run(shlexf("groupadd --system %s && useradd --system --gid %s --shell /bin/bash --create-home %s", opts.User, opts.User, opts.User)).
			User(opts.User)
	}

	state = steamcmdState.
		// `echo`ing the buildid here is to workaround
		// buildkit cacheing the steamcmd app_update command
		// when there has been a new build pushed to the branch.
		Run(shlexf("echo %d && steamcmd %s", buildID, strings.Join(arg, " "))).
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

func getSolveOpt(ctx context.Context, appID int, exportType string, output io.WriteCloser, opts *BuildImageOpts) (*client.SolveOpt, int, error) {
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
				Type: exportType,
				Attrs: map[string]string{
					exptypes.ExporterImageConfigKey: string(ib),
				},
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return output, nil
				},
			},
		},
	}, buildID, nil
}

func newBuildImageOpts(opts ...BuildImageOpt) *BuildImageOpts {
	o := &BuildImageOpts{
		BaseImageRef:     DefaultBaseImageRef,
		SteamcmdImageRef: DefaultSteamcmdImageRef,
		Dir:              DefaultDir,
		LaunchType:       DefaultLaunchType,
		PlatformType:     steamcmd.PlatformTypeLinux,
		User:             DefaultUser,
	}

	for _, opt := range opts {
		opt.Apply(o)
	}

	return o
}

func shlexf(format string, a ...any) llb.RunOption {
	if strings.Contains(format, " && ") {
		return llb.Shlex("sh -c '" + fmt.Sprintf(format, a...) + "'")
	}

	return llb.Shlexf(format, a...)
}

var (
	errManifestFound = errors.New("manifest found")
)

func getImageManifest(ctx context.Context, appID int, a *ImageBuilder, opts ...BuildImageOpt) (*v1.Manifest, error) {
	var (
		_        = log.FromContext(ctx)
		pr, pw   = io.Pipe()
		manifest = &v1.Manifest{}
	)
	defer pr.Close()

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return a.BuildImage(ctx, appID, pw, opts...)
	})

	eg.Go(func() error {
		tr := tar.NewReader(pr)

		for {
			if _, err := tr.Next(); errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return err
			}

			if err := json.NewDecoder(tr).Decode(manifest); err == nil {
				return errManifestFound
			}
		}

		return fmt.Errorf("manifest not found")
	})

	if err := eg.Wait(); errors.Is(err, errManifestFound) {
		return manifest, nil
	} else if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("manifest not found")
}

func (a *ImageBuilder) BuildImage(ctx context.Context, appID int, output io.WriteCloser, opts ...BuildImageOpt) error {
	var (
		_ = log.FromContext(ctx)
		o = newBuildImageOpts(opts...)
	)

	solvOpt, buildID, err := getSolveOpt(ctx, appID, client.ExporterDocker, output, o)
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
		errC := make(chan error, 1)

		// Solve doesn't seem to return when context is cancelled,
		// so we have to wait on the context ourselves.
		go func() {
			_, err := a.Solve(ctx, def, *solvOpt, solvStatusC)
			errC <- err
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errC:
			return err
		}
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
