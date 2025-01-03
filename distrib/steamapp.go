package distrib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/frantjc/sindri/distrib/cache"
	"github.com/frantjc/sindri/internal/img"
	"github.com/frantjc/sindri/steamapp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

type SteamappPuller struct {
	Base               v1.Image
	Dir                string
	Username, Password string
	User, Group        string
	Store              cache.Store
}

func (p *SteamappPuller) HeadManifest(_ context.Context, name string, _ string) error {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if appID <= 0 {
		return fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	return nil
}

var errValNotFound = errors.New("value not found")

func (p *SteamappPuller) getManifest(appID int, reference string) (*Manifest, error) {
	if err := p.init(); err != nil {
		return nil, err
	}

	manifest := &Manifest{}
	if ok, err := p.Store.Get(fmt.Sprintf("steamapp:%d::ref:%s", appID, reference), manifest); err == nil && ok {
		return manifest, nil
	} else if err != nil {
		return nil, err
	}

	return nil, errValNotFound
}

func (p *SteamappPuller) getBranch(appID int, digest string) (string, error) {
	if err := p.init(); err != nil {
		return "", err
	}

	branch := ""
	if ok, err := p.Store.Get(fmt.Sprintf("steamapp:%d::digest:%s", appID, digest), &branch); err == nil && ok {
		return branch, nil
	} else if err != nil {
		return "", err
	}

	return "", errValNotFound
}

func (p *SteamappPuller) setManifest(appID int, reference string, manifest *Manifest) error {
	if err := p.init(); err != nil {
		return err
	}

	return p.Store.Set(fmt.Sprintf("steamapp:%d::ref:%s", appID, reference), manifest)
}

func (p *SteamappPuller) init() error {
	if p.Store == nil {
		var err error
		p.Store, err = cache.NewStore("mem://")
		return err
	}
	return nil
}

func (p *SteamappPuller) setBranch(appID int, digest, reference string) error {
	if err := p.init(); err != nil {
		return err
	}

	return p.Store.Set(fmt.Sprintf("steamapp:%d::digest:%s", appID, digest), struct {
		Branch string `json:"branch"`
	}{reference})
}

func (p *SteamappPuller) GetManifest(ctx context.Context, name string, reference string) (*Manifest, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if appID <= 0 {
		return nil, fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	if reference == "latest" {
		reference = "public"
	}

	if manifest, err := p.getManifest(appID, reference); err == nil {
		return manifest, nil
	} else if !errors.Is(err, errValNotFound) {
		return nil, err
	}

	image := p.Base

	cfg, err := p.getConfig(ctx, image, appID, reference)
	if err != nil {
		return nil, err
	}

	image, err = mutate.Config(image, *cfg)
	if err != nil {
		return nil, err
	}

	layer, err := p.getLayer(ctx, appID, reference)
	if err != nil {
		return nil, err
	}

	image, err = mutate.AppendLayers(image, layer)
	if err != nil {
		return nil, err
	}

	manifest, err := image.Manifest()
	if err != nil {
		return nil, err
	}

	digest := manifest.Config.Digest.String()

	if err = p.setManifest(appID, digest, manifest); err != nil {
		return nil, err
	}

	if err = p.setBranch(appID, digest, reference); err != nil {
		return nil, err
	}

	return manifest, nil
}

func (p *SteamappPuller) HeadBlob(_ context.Context, name string, _ string) error {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if appID <= 0 {
		return fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	return nil
}

func (p *SteamappPuller) GetBlob(ctx context.Context, name string, digest string) (Blob, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if appID <= 0 {
		return nil, fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	image := p.Base

	if branch, err := p.getBranch(appID, digest); err == nil {
		layer, err := p.getLayer(ctx, appID, branch)
		if err != nil {
			return nil, err
		}

		image, err = mutate.AppendLayers(image, layer)
		if err != nil {
			return nil, err
		}

		cfg, err := p.getConfig(ctx, image, appID, branch)
		if err != nil {
			return nil, err
		}

		image, err = mutate.Config(image, *cfg)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, errValNotFound) {
		return nil, err
	}

	hash, err := v1.NewHash(digest)
	if err != nil {
		return nil, err
	}

	return image.LayerByDigest(hash)
}

func (p *SteamappPuller) getLayer(ctx context.Context, appID int, branch string) (Blob, error) {
	layer, err := img.ReproducibleBuildLayerInDirFromOpener(
		func() (io.ReadCloser, error) {
			return steamapp.Open(ctx, appID, steamapp.WithAccount(p.Username, p.Password), steamapp.WithBeta(branch, ""))
		},
		p.Dir,
		p.User,
		p.Group,
	)
	if err != nil {
		return nil, err
	}

	hash, err := layer.Digest()
	if err != nil {
		return nil, err
	}

	if err = p.setBranch(appID, hash.String(), branch); err != nil {
		return nil, err
	}

	return layer, nil
}

func (p *SteamappPuller) getConfig(ctx context.Context, image v1.Image, appID int, branch string) (*v1.Config, error) {
	cfgf, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}

	return steamapp.ImageConfig(
		ctx, appID, &cfgf.Config,
		steamapp.WithAccount(p.Username, p.Password),
		steamapp.WithInstallDir(p.Dir),
		steamapp.WithBeta(branch, ""),
		steamapp.WithLaunchType("server"),
	)
}
