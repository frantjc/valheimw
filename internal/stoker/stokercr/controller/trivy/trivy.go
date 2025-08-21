package trivy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/commands/operation"
	"github.com/aquasecurity/trivy/pkg/db"
	fanaltypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/flag"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/frantjc/sindri/internal/cache"
	"github.com/frantjc/sindri/internal/logutil"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/google/go-containerregistry/pkg/name"
)

type Scanner struct {
	DBRepositories []name.Reference
	CacheDir       string

	runner artifact.Runner
}

type ScannerOpts struct {
	DBRepositories []string
	CacheDir       string
}

func (o *ScannerOpts) Apply(opts *ScannerOpts) {
	if o != nil {
		if opts != nil {
			opts.DBRepositories = append(opts.DBRepositories, o.DBRepositories...)
		}

		if o.CacheDir != "" {
			opts.CacheDir = o.CacheDir
		}
	}
}

type ScannerOpt interface {
	Apply(*ScannerOpts)
}

func WithDBRepositories(repos []string) ScannerOpt {
	return &ScannerOpts{DBRepositories: repos}
}

func NewScanner(ctx context.Context, opts ...ScannerOpt) (*Scanner, error) {
	o := &ScannerOpts{
		DBRepositories: []string{
			"ghcr.io/aquasecurity/trivy-db:2",
			"mirror.gcr.io/aquasec/trivy-db:2",
		},
	}

	for _, opt := range opts {
		opt.Apply(o)
	}

	repoRefs := make([]name.Reference, 0, len(o.DBRepositories))
	for _, repo := range o.DBRepositories {
		ref, err := name.ParseReference(repo)
		if err != nil {
			return nil, fmt.Errorf("parse vuln db repository %s: %w", repo, err)
		}
		repoRefs = append(repoRefs, ref)
	}

	cacheDir := filepath.Join(cache.Dir, "trivy")

	if err := operation.DownloadDB(
		ctx,
		"dev",
		cacheDir,
		repoRefs,
		false,
		false,
		fanaltypes.RegistryOptions{},
	); err != nil {
		return nil, fmt.Errorf("download trivy db: %w", err)
	}

	if err := db.Init(db.Dir(cacheDir)); err != nil {
		return nil, fmt.Errorf("initialize trivy db: %w", err)
	}

	runner, err := artifact.NewRunner(
		ctx,
		flag.Options{},
		artifact.TargetContainerImage,
	)
	if err != nil {
		return nil, fmt.Errorf("create trivy runner: %w", err)
	}

	return &Scanner{repoRefs, cacheDir, runner}, nil
}

func (s *Scanner) Scan(ctx context.Context, r io.Reader) ([]v1alpha1.Vulnerability, error) {
	log := logutil.SloggerFrom(ctx)

	if f, ok := r.(*os.File); ok {
		log.Debug("skipping intermediate file write", "file", f.Name())

		return s.scanFile(ctx, f.Name())
	}

	if err := os.MkdirAll(s.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create dir for image tar: %w", err)
	}

	f, err := os.CreateTemp(s.CacheDir, "*.tar")
	if err != nil {
		return nil, fmt.Errorf("write image tar: %w", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if _, err = io.Copy(f, r); err != nil {
		return nil, err
	}

	return s.scanFile(ctx, f.Name())
}

func (s *Scanner) scanFile(ctx context.Context, p string) ([]v1alpha1.Vulnerability, error) {
	var (
		log   = logutil.SloggerFrom(ctx)
		debug = log.Enabled(ctx, slog.LevelDebug)
	)

	log.Debug("calling trivy image scanner")

	rep, err := s.runner.ScanImage(ctx, flag.Options{
		GlobalOptions: flag.GlobalOptions{
			CacheDir: s.CacheDir,
			Quiet:    !debug,
			Debug:    debug,
			Timeout:  5 * time.Minute,
		},
		DBOptions: flag.DBOptions{
			SkipDBUpdate:   false,
			DownloadDBOnly: false,
			DBRepositories: s.DBRepositories,
		},
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
		return nil, fmt.Errorf("scan image with trivy: %w", err)
	}

	vulns := []v1alpha1.Vulnerability{}

	for _, res := range rep.Results {
		for _, v := range res.Vulnerabilities {
			vulns = append(vulns, v1alpha1.Vulnerability{
				ID:        v.VulnerabilityID,
				PackageID: v.PkgID,
				Title:     v.Title,
				Severity:  v.Severity,
				Status:    v.Status.String(),
			})
		}
	}

	return vulns, nil
}
