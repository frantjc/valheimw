// A generated module for Valheimw functions

package main

import (
	"context"
	"strings"

	"github.com/frantjc/valheimw/.dagger/internal/dagger"
	xslices "github.com/frantjc/x/slices"
)

type ValheimwDev struct {
	Source *dagger.Directory
}

func New(
	ctx context.Context,
	// +optional
	// +defaultPath="."
	src *dagger.Directory,
) (*ValheimwDev, error) {
	return &ValheimwDev{
		Source: src,
	}, nil
}

func (m *ValheimwDev) Fmt(ctx context.Context) *dagger.Changeset {
	goModules := []string{
		".dagger/",
	}

	root := dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: goModules,
		}),
	}).
		Container().
		WithExec([]string{"go", "fmt", "./..."}).
		Directory(".")

	for _, module := range goModules {
		root = root.WithDirectory(
			module,
			dag.Go(dagger.GoOpts{
				Module: m.Source.Directory(module).Filter(dagger.DirectoryFilterOpts{
					Exclude: xslices.Filter(goModules, func(m string, _ int) bool {
						return strings.HasPrefix(m, module)
					}),
				}),
			}).
				Container().
				WithExec([]string{"go", "fmt", "./..."}).
				Directory("."),
		)
	}

	return root.Changes(m.Source)
}

const (
	gid   = "1001"
	uid   = gid
	group = "valheimw"
	user  = group
	owner = user + ":" + group
	home  = "/home/" + user
)

func (m *ValheimwDev) Container(ctx context.Context) (*dagger.Container, error) {
	return dag.Container().From("debian:stable-slim").
		WithExec([]string{"apt-get", "update", "-y"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "lib32gcc-s1"}).
		WithExec([]string{"rm", "-rf", "/var/lib/apt/lists/*"}).
		WithExec([]string{"apt-get", "clean"}).
		WithExec([]string{"groupadd", "-r", "-g", gid, group}).
		WithExec([]string{"useradd", "-m", "-g", group, "-u", uid, "-r", user}).
		WithEnvVariable("PATH", home+"/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithFile(
			home+"/.local/bin/valheimw", m.Binary(ctx),
			dagger.ContainerWithFileOpts{Expand: true, Owner: owner, Permissions: 0700}).
		WithExec([]string{"chown", "-R", owner, home}).
		WithUser(user).
		WithWorkdir(home).
		WithEntrypoint([]string{"valheimw"}), nil
}

func (m *ValheimwDev) Service(ctx context.Context) (*dagger.Service, error) {
	container, err := m.Container(ctx)
	if err != nil {
		return nil, err
	}

	return container.
		WithExposedPort(2456, dagger.ContainerWithExposedPortOpts{
			// Protocol: dagger.NetworkProtocolUdp,
			ExperimentalSkipHealthcheck: true,
		}).
		WithExposedPort(8080).
		WithSecretVariable("VALHEIM_PASSWORD", dag.SetSecret("VALHEIM_PASSWORD", "plaintext")).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint:                 true,
			Args: []string{
				"--debug",
				"--mod-category-check",
				"--mod=Advize/PlantEasily",
				"--mod=shudnal/ExtraSlots",
				"--mod=Goldenrevolver/Quick_Stack_Store_Sort_Trash_Restock",
				"--mod=Smoothbrain/TargetPortal",
			},
		}), nil
}

func (m *ValheimwDev) Version(ctx context.Context) string {
	version := "v0.0.0-unknown"

	gitRef := m.Source.AsGit().LatestVersion()

	if ref, err := gitRef.Ref(ctx); err == nil {
		version = strings.TrimPrefix(ref, "refs/tags/")
	}

	if latestVersionCommit, err := gitRef.Commit(ctx); err == nil {
		if headCommit, err := m.Source.AsGit().Head().Commit(ctx); err == nil {
			if headCommit != latestVersionCommit {
				if len(headCommit) > 7 {
					headCommit = headCommit[:7]
				}
				version += "-" + headCommit
			}
		}
	}

	if empty, _ := m.Source.AsGit().Uncommitted().IsEmpty(ctx); !empty {
		version += "+dirty"
	}

	return version
}

func (m *ValheimwDev) Tag(ctx context.Context) string {
	before, _, _ := strings.Cut(strings.TrimPrefix(m.Version(ctx), "v"), "+")
	return before
}

func (m *ValheimwDev) Binary(ctx context.Context) *dagger.File {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{".github/", "e2e/"},
		}),
	}).
		Build(dagger.GoBuildOpts{
			Pkg:     "./cmd/valheimw",
			Ldflags: "-s -w -X main.version=" + m.Version(ctx),
		})
}

func (m *ValheimwDev) Vulncheck(ctx context.Context) (string, error) {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{
				".dagger/",
			},
		}),
	}).
		Container().
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@v1.1.4"}).
		WithExec([]string{"govulncheck", "./..."}).
		CombinedOutput(ctx)
}

func (m *ValheimwDev) Vet(ctx context.Context) (string, error) {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{
				".dagger/",
			},
		}),
	}).
		Container().
		WithExec([]string{"go", "vet", "./..."}).
		CombinedOutput(ctx)
}

func (m *ValheimwDev) Staticcheck(ctx context.Context) (string, error) {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{
				".dagger/",
			},
		}),
	}).
		Container().
		WithExec([]string{"go", "install", "honnef.co/go/tools/cmd/staticcheck@v0.6.1"}).
		WithExec([]string{"staticcheck", "./..."}).
		CombinedOutput(ctx)
}

func (m *ValheimwDev) Coder(ctx context.Context) (*dagger.LLM, error) {
	gopls := dag.Go(dagger.GoOpts{Module: m.Source}).
		Container().
		WithExec([]string{"go", "install", "golang.org/x/tools/gopls@latest"})

	instructions, err := gopls.WithExec([]string{"gopls", "mcp", "-instructions"}).Stdout(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Doug().
		Agent(
			dag.LLM().
				WithEnv(
					dag.Env().
						WithCurrentModule().
						WithWorkspace(m.Source.Filter(dagger.DirectoryFilterOpts{
							Exclude: []string{".dagger/", ".github/"},
						})),
				).
				WithBlockedFunction("ValheimwDev", "container").
				WithBlockedFunction("ValheimwDev", "service").
				WithBlockedFunction("ValheimwDev", "tag").
				WithBlockedFunction("ValheimwDev", "version").
				WithSystemPrompt(instructions).
				WithMCPServer(
					"gopls",
					gopls.AsService(dagger.ContainerAsServiceOpts{
						Args: []string{"gopls", "mcp"},
					}),
				),
		), nil
}
