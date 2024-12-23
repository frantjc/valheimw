package distrib

import (
	"context"
	"fmt"
	"io"
	"strconv"

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

	steamappIDDigestToBranch   map[int]map[string]string
	steamappIDDigestToManifest map[int]map[string]Manifest
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

	if p.steamappIDDigestToManifest == nil {
		p.steamappIDDigestToManifest = map[int]map[string]Manifest{}
	}

	if referenceToManifest, ok := p.steamappIDDigestToManifest[appID]; !ok {
		p.steamappIDDigestToManifest[appID] = map[string]v1.Manifest{}
	} else if manifest, ok := referenceToManifest[reference]; ok {
		return &manifest, nil
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

	p.steamappIDDigestToManifest[appID][manifest.Config.Digest.String()] = *manifest
	p.steamappIDDigestToBranch[appID][manifest.Config.Digest.String()] = reference

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

	if digestToBranch, ok := p.steamappIDDigestToBranch[appID]; ok {
		if branch, ok := digestToBranch[digest]; ok {
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
		}
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

	if p.steamappIDDigestToBranch == nil {
		p.steamappIDDigestToBranch = map[int]map[string]string{}
	}

	if p.steamappIDDigestToBranch[appID] == nil {
		p.steamappIDDigestToBranch[appID] = map[string]string{}
	}

	p.steamappIDDigestToBranch[appID][hash.String()] = branch

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
