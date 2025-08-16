package scanners

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aquasecurity/trivy/pkg/cache"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/commands/operation"
	"github.com/aquasecurity/trivy/pkg/db"
	fanaltypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/flag"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/frantjc/sindri/internal/stoker/stokercr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/uuid"
)

type trivy struct {
	runner artifact.Runner
}

type TrivyOptions struct {
	DBRepositories []string
}

type TrivyOption func(*TrivyOptions)

func WithDBRepositories(repos []string) TrivyOption {
	return func(o *TrivyOptions) {
		o.DBRepositories = repos
	}
}

func NewTrivy(ctx context.Context, opts ...TrivyOption) (*trivy, error) {
	options := &TrivyOptions{}

	for _, opt := range opts {
		opt(options)
	}

	if len(options.DBRepositories) == 0 {
		options.DBRepositories = []string{
			"ghcr.io/aquasecurity/trivy-db:2",
			"mirror.gcr.io/aquasec/trivy-db:2",
		}
	}

	dbRepos := make([]name.Reference, 0, len(options.DBRepositories))
	for _, repo := range options.DBRepositories {
		ref, err := name.ParseReference(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repository %s: %w", repo, err)
		}
		dbRepos = append(dbRepos, ref)
	}

	if err := operation.DownloadDB(
		ctx,
		"dev",
		cache.DefaultDir(),
		dbRepos,
		false,
		false,
		fanaltypes.RegistryOptions{},
	); err != nil {
		return nil, fmt.Errorf("failed to download Trivy DB: %w", err)
	}

	if err := db.Init(db.Dir(cache.DefaultDir())); err != nil {
		return nil, fmt.Errorf("failed to initialize Trivy DB: %w", err)
	}

	runner, err := artifact.NewRunner(
		ctx,
		flag.Options{
			GlobalOptions: flag.GlobalOptions{
				CacheDir: cache.DefaultDir(),
				Quiet:    true,
				Debug:    false,
				Timeout:  5 * time.Minute,
			},
			DBOptions: flag.DBOptions{
				SkipDBUpdate:   false,
				DownloadDBOnly: false,
				DBRepositories: dbRepos,
			},
		},
		artifact.TargetContainerImage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Trivy runner: %w", err)
	}

	return &trivy{runner}, nil
}

func (s trivy) Scan(ctx context.Context, b bytes.Buffer) ([]stokercr.Vuln, error) {
	p := fmt.Sprintf("%s/sindri/image-%s.tar", cache.DefaultDir(), uuid.New())

	if err := os.WriteFile(p, b.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write image tar: %w", err)
	}
	defer os.Remove(p)

	return s.scanFile(ctx, p)
}

func (s trivy) scanFile(ctx context.Context, p string) ([]stokercr.Vuln, error) {
	rep, err := s.runner.ScanImage(ctx, flag.Options{
		ScanOptions: flag.ScanOptions{
			Target:   p,
			Scanners: types.Scanners{types.VulnerabilityScanner},
		},
		ImageOptions: flag.ImageOptions{
			Input:               p,
			ImageConfigScanners: types.Scanners{types.VulnerabilityScanner},
		},
		PackageOptions: flag.PackageOptions{
			PkgTypes:         []string{types.PkgTypeOS, types.PkgTypeLibrary},
			PkgRelationships: fanaltypes.Relationships,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed scan image with Trivy: %w", err)
	}

	var vulns []stokercr.Vuln
	for _, res := range rep.Results {
		for _, v := range res.Vulnerabilities {
			vulns = append(vulns, stokercr.Vuln{
				ID:          v.VulnerabilityID,
				PackageID:   v.PkgID,
				Title:       v.Title,
				Description: v.Description,
				Severity:    stokercr.NewSeverity(v.Severity),
				Status:      stokercr.VulnStatus(v.Status),
			})
		}
	}

	return vulns, nil
}
