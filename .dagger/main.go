// A generated module for Valheimw functions

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/frantjc/valheimw/.dagger/internal/dagger"
)

type ValheimwDev struct {}

const (
	gid   = "1001"
	uid   = gid
	group = "valheimw"
	user  = group
	owner = user + ":" + group
	home  = "/home/" + user
)

func (m *ValheimwDev) Container(ctx context.Context, ws *dagger.Workspace) (*dagger.Container, error) {
	return dag.Container().From("debian:stable-slim").
		WithExec([]string{"apt-get", "update", "-y"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "lib32gcc-s1"}).
		WithExec([]string{"rm", "-rf", "/var/lib/apt/lists/*"}).
		WithExec([]string{"apt-get", "clean"}).
		WithExec([]string{"groupadd", "-r", "-g", gid, group}).
		WithExec([]string{"useradd", "-m", "-g", group, "-u", uid, "-r", user}).
		WithEnvVariable("PATH", home+"/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithFile(
			home+"/.local/bin/valheimw", m.Binary(ctx, ws),
			dagger.ContainerWithFileOpts{Expand: true, Owner: owner, Permissions: 0700}).
		WithExec([]string{"chown", "-R", owner, home}).
		WithUser(user).
		WithWorkdir(home).
		WithEntrypoint([]string{"valheimw"}), nil
}

func (m *ValheimwDev) Service(ctx context.Context, ws *dagger.Workspace) (*dagger.Service, error) {
	container, err := m.Container(ctx, ws)
	if err != nil {
		return nil, err
	}

	return container.
		WithExposedPort(8080).
		WithSecretVariable("VALHEIM_PASSWORD", dag.SetSecret("VALHEIM_PASSWORD", "plaintext")).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint: true,
			Args: []string{
				"--debug",
				"--no-valheim",
				"--mod=Advize/PlantEasily",
				"--mod=shudnal/ExtraSlots",
				"--mod=Goldenrevolver/Quick_Stack_Store_Sort_Trash_Restock?category=Client-side",
			},
		}), nil
}

func (m *ValheimwDev) Version(ctx context.Context, ws *dagger.Workspace) string {
	version := "v0.0.0-unknown"

	src := ws.Directory(".")
	gitRepo := src.AsGit()
	gitRef := gitRepo.LatestVersion()

	if ref, err := gitRef.Ref(ctx); err == nil {
		version = strings.TrimPrefix(ref, "refs/tags/")
	}

	if latestVersionCommit, err := gitRef.Commit(ctx); err == nil {
		if headCommit, err := gitRepo.Head().Commit(ctx); err == nil {
			if headCommit != latestVersionCommit {
				if len(headCommit) > 7 {
					headCommit = headCommit[:7]
				}
				version += "-" + headCommit
			}
		}
	}

	if empty, _ := gitRepo.Uncommitted().IsEmpty(ctx); !empty {
		version += "+dirty"
	}

	return version
}

func (m *ValheimwDev) Tag(ctx context.Context, ws *dagger.Workspace) string {
	before, _, _ := strings.Cut(strings.TrimPrefix(m.Version(ctx, ws), "v"), "+")
	return before
}

func (m *ValheimwDev) Binary(ctx context.Context, ws *dagger.Workspace) *dagger.File {
	return dag.Go(dagger.GoOpts{
		Ws: ws,
	}).
		Build(dagger.GoBuildOpts{
			Pkg:     "./cmd/valheimw",
			Ldflags: "-s -w -X main.version=" + m.Version(ctx, ws),
		})
}

func (m *ValheimwDev) Release(
	ctx context.Context,
	ws *dagger.Workspace,
	githubToken *dagger.Secret,
	// +optional
	githubRepo string,
) error {
	_, repo, ok := strings.Cut(githubRepo, "/")
	if !ok {
		return fmt.Errorf("expected org/repo format, got %q", githubRepo)
	}

	gh := dag.Gh(githubToken)
	src := ws.Directory(".", dagger.WorkspaceDirectoryOpts{
		Gitignore: true,
	})

	gitRepository := src.AsGit()
	latestVersion := gitRepository.LatestVersion()

	ref, err := latestVersion.Ref(ctx)
	if err != nil {
		return err
	}
	tag := strings.TrimPrefix(ref, "refs/tags/")


	bin := dag.Upx().Pack(m.Binary(ctx, ws))
	file := fmt.Sprintf("%s-%s-linux-amd64.tar.gz", repo, tag,)
	asset := dag.Archive().
		Tar(
			src.Filter(dagger.DirectoryFilterOpts{
				Include: []string{
					"README.md",
					"LICENSE",
				},
			}).
				WithFile(
					repo,
					bin,
				),
			dagger.ArchiveTarOpts{
				Gzip: true,
			},
		).WithName(file)

	assets := []*dagger.File{asset}

	container, err := m.Container(ctx, ws)
	if err != nil {
		return err
	}

	release := gh.Release(githubRepo, tag)

	if err := release.Create(ctx, dagger.GhReleaseCreateOpts{
		Draft:         true,
		GenerateNotes: true,
	}); err != nil {
		return err
	}

	if err := release.Upload(ctx, assets, dagger.GhReleaseUploadOpts{
		Clobber: true,
	}); err != nil {
		return err
	}

	registry := "ghcr.io"
	container = container.WithRegistryAuth(registry, "x-access-token", githubToken)

	if _, err := container.Publish(ctx, fmt.Sprintf("%s/%s:%s", registry, githubRepo, m.Tag(ctx, ws))); err != nil {
		return err
	}

	if _, err := container.Publish(ctx, fmt.Sprintf("%s/%s:latest", registry, githubRepo)); err != nil {
		return err
	}

	if err := release.Edit(ctx, dagger.GhReleaseEditOpts{
		Latest: true,
	}); err != nil {
		return err
	}

	return nil
}
