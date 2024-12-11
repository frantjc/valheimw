package sindri

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/frantjc/sindri/internal/cache"
	"github.com/frantjc/sindri/internal/layerutil"
	xslice "github.com/frantjc/x/slice"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/opencontainers/go-digest"
)

const (
	DefaultInstalledLabelPrefix = "cc.frantj.sindri"
	ImageReference              = "sindri.frantj.cc"
)

type Manager struct {
	dbPath               string
	installedLabelPrefix string
	img v1.Image
	inMem bool

	mu sync.Mutex
}

type ManagerOpt func(*Manager)

func WithDBPath(path string) ManagerOpt {
	return func(m *Manager) {
		m.dbPath = path
	}
}

var (
	tag = func() *name.Tag {
		t, err := name.NewTag(ImageReference)
		if err != nil {
			panic(err)
		}

		return &t
	}()
	ref = name.MustParseReference(ImageReference)
)

func defaultManager() *Manager {
	return &Manager{
		dbPath:               filepath.Join(cache.Dir, "sindri.db"),
		installedLabelPrefix: DefaultInstalledLabelPrefix,
		inMem: true,
	}
}

func NewManager(opts ...ManagerOpt) *Manager {
	mgr := defaultManager()

	for _, opt := range opts {
		opt(mgr)
	}

	return mgr
}

func (m *Manager) parseInstallationLabel(key, value string) (*Installation, error) {
	parts := strings.Split(
		strings.TrimPrefix(key, fmt.Sprintf("%s.", m.installedLabelPrefix)),
		".",
	)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid layer digest label: %s", key)
	}

	u, err := url.Parse(value)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	dir := q.Get("dir")
	q.Del("dir")
	u.RawQuery = q.Encode()

	return &Installation{
		Digest:   digest.NewDigestFromHex(digest.Canonical.String(), parts[0]),
		CacheKey: strings.Join(parts[1:], "."),
		URL:      u,
		Dir:      dir,
	}, nil
}

type Installation struct {
	Digest   digest.Digest
	URL      *url.URL
	CacheKey string
	Dir      string
}

func (m *Manager) installationLabelKey(hex, cacheKey string) string {
	return strings.Join([]string{m.installedLabelPrefix, hex, cacheKey}, ".")
}

func (m *Manager) installationLabelValue(u *url.URL, dir string) string {
	u0 := u.JoinPath() // Copy the URL.
	q := u.Query()
	q.Add("dir", dir)

	for k, v := range q {
		if len(xslice.Filter(v, func(w string, _ int) bool {
			return w != ""
		})) == 0 {
			q.Del(k)
		}
	}

	u0.RawQuery = q.Encode()

	return u0.String()
}

func (m *Manager) load() (v1.Image, error) {
	if m.inMem && m.img != nil {
		return m.img, nil
	}

	img := empty.Image

	if fi, err := os.Stat(m.dbPath); err == nil && !fi.IsDir() {
		if img, err = tarball.ImageFromPath(m.dbPath, tag); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if m.inMem {
		m.img = img
	}

	return img, nil
}

func (m *Manager) ExtractAll() (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	img, err := m.load()
	if err != nil {
		return nil, err
	}

	return mutate.Extract(img), nil
}

func (m *Manager) Extract(scheme string, schemes ...string) (io.ReadCloser, error) {
	schemes = xslice.Filter(append(schemes, scheme), func(s string, _ int) bool {
		return s != ""
	})
	if len(schemes) == 0 {
		return nil, fmt.Errorf("no non-empty schemes provided")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	img, err := m.load()
	if err != nil {
		return nil, err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	hexes := []string{}

	for k, v := range cfg.Config.Labels {
		if strings.HasPrefix(k, m.installedLabelPrefix) {
			installation, err := m.parseInstallationLabel(k, v)
			if err != nil {
				return nil, err
			}

			if xslice.Includes(schemes, installation.URL.Scheme) {
				hexes = append(hexes, installation.Digest.Hex())
			}
		}
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	filteredLayers := []v1.Layer{}

	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return nil, err
		}

		if xslice.Includes(hexes, digest.Hex) {
			filteredLayers = append(filteredLayers, layer)
		}
	}

	img, err = mutate.AppendLayers(empty.Image, filteredLayers...)
	if err != nil {
		return nil, err
	}

	return mutate.Extract(img), nil
}

func (m *Manager) Install(ctx context.Context, installableURL, dir string) (*Installation, error) {
	dir = strings.TrimPrefix(dir, "/")

	u, err := url.Parse(installableURL)
	if err != nil {
		return nil, err
	}

	labelValue := m.installationLabelValue(u, dir)

	m.mu.Lock()
	defer m.mu.Unlock()

	img, err := m.load()
	if err != nil {
		return nil, err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	cacheKey, err := CacheKey(ctx, installableURL)
	if err != nil {
		return nil, err
	}

	for k, v := range cfg.Config.Labels {
		if v == labelValue {
			installed, err := m.parseInstallationLabel(k, v)
			if err != nil {
				return nil, err
			}

			if installed.CacheKey == cacheKey {
				return installed, nil
			}

			layers, err := img.Layers()
			if err != nil {
				return nil, err
			}

			filteredLayers := []v1.Layer{}

			for _, layer := range layers {
				digest, err := layer.Digest()
				if err != nil {
					return nil, err
				}

				if digest.Hex != installed.Digest.Hex() {
					filteredLayers = append(filteredLayers, layer)
				}
			}

			img, err = mutate.AppendLayers(empty.Image, filteredLayers...)
			if err != nil {
				return nil, err
			}

			delete(cfg.Config.Labels, k)
		}
	}

	layer, err := layerutil.ReproducibleBuildLayerInDirFromOpener(func() (io.ReadCloser, error) {
		return Open(ctx, installableURL)
	}, dir)
	if err != nil {
		return nil, err
	}

	hash, err := layer.Digest()
	if err != nil {
		return nil, err
	}

	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return nil, err
	}

	if cfg.Config.Labels == nil {
		cfg.Config.Labels = map[string]string{}
	}

	installedLabel := m.installationLabelKey(hash.Hex, cacheKey)

	cfg.Config.Labels[installedLabel] = labelValue

	img, err = mutate.Config(img, cfg.Config)
	if err != nil {
		return nil, err
	}

	if err = m.save(img); err != nil {
		return nil, err
	}

	installation, err := m.parseInstallationLabel(installedLabel, installableURL)
	if err != nil {
		return nil, err
	}

	return installation, nil
}

func (m *Manager) Uninstall(ctx context.Context, installableURL, dir string) error {
	u, err := url.Parse(installableURL)
	if err != nil {
		return err
	}

	labelValue := m.installationLabelValue(u, dir)

	m.mu.Lock()
	defer m.mu.Unlock()

	img, err := m.load()
	if err != nil {
		return err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return err
	}

	for k, v := range cfg.Config.Labels {
		if strings.HasPrefix(k, m.installedLabelPrefix) {
			installation, err := m.parseInstallationLabel(k, v)
			if err != nil {
				return err
			}

			if installation.URL.String() == labelValue {
				img, err := m.load()
				if err != nil {
					return err
				}

				layers, err := img.Layers()
				if err != nil {
					return err
				}

				filteredLayers := []v1.Layer{}

				for _, layer := range layers {
					digest, err := layer.Digest()
					if err != nil {
						return err
					}

					if digest.Hex != installation.Digest.Hex() {
						filteredLayers = append(filteredLayers, layer)
					}
				}

				img, err = mutate.AppendLayers(empty.Image, filteredLayers...)
				if err != nil {
					return err
				}

				delete(cfg.Config.Labels, k)

				img, err = mutate.Config(img, cfg.Config)
				if err != nil {
					return err
				}

				if err = m.save(img); err != nil {
					return err
				}

				return nil
			}
		}
	}

	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.inMem = false
	defer func() {
		m.inMem = true
	}()

	return m.save(m.img)
}

func (m *Manager) save(img v1.Image) error {
	if m.inMem {
		m.img = img
	}

	dir, file := filepath.Split(m.dbPath)

	tmp, err := os.CreateTemp(dir, fmt.Sprintf("%s.*", file))
	if err != nil {
		return err
	}
	defer tmp.Close()
	defer os.Remove(tmp.Name())

	if err = tarball.Write(ref, img, tmp); err != nil {
		return err
	}

	if err = tmp.Close(); err != nil {
		return err
	}

	if err = os.Rename(tmp.Name(), m.dbPath); err != nil {
		return err
	}

	return nil
}
