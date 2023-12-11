package sindri

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/frantjc/go-fn"
	"github.com/frantjc/sindri/steamcmd"
	"github.com/frantjc/sindri/thunderstore"
	xcontainerregistry "github.com/frantjc/sindri/x/containerregistry"
	xtar "github.com/frantjc/sindri/x/tar"
	xzip "github.com/frantjc/sindri/x/zip"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// ModMetadata stores metadata about an added mod.
type ModMetadata struct {
	LayerDigest string `json:"layerDigest,omitempty"`
	Version     string `json:"version,omitempty"`
}

// Metadata stores metadata about a downloaded game
// and added mods.
type Metadata struct {
	BaseLayerDigest string                 `json:"baseLayerDigest,omitempty"`
	Mods            map[string]ModMetadata `json:"mods,omitempty"`
}

// Sindri manages the files of a game and its mods.
type Sindri struct {
	SteamAppID         string
	BepInEx            *thunderstore.Package
	ThunderstoreClient *thunderstore.Client

	mu                 *sync.Mutex
	stateDir, rootDir  string
	img                v1.Image
	tag                *name.Tag
	metadata           *Metadata
	initialized        bool
	beta, betaPassword string
}

// Opt is an option to pass when creating
// a new Sindri instance.
type Opt func(*Sindri)

// WithRootDir sets a *Sindri's root directory
// where it will store any persistent data.
func WithRootDir(dir string) Opt {
	return func(s *Sindri) {
		s.rootDir = dir
	}
}

// WithStateDir sets a *Sindri's state directory
// where it will store any ephemeral data.
func WithStateDir(dir string) Opt {
	return func(s *Sindri) {
		s.stateDir = dir
	}
}

// WithBeta makes Sindri use the given Steam beta.
func WithBeta(beta string, password string) Opt {
	return func(s *Sindri) {
		s.beta = beta
		s.betaPassword = password
	}
}

const (
	// ImageRef is the image reference that Sindri
	// stores a game and its mods' files at inside
	// of it's .tar file.
	ImageRef = "frantj.cc/sindri"
)

// New creates a new Sindri instance with the given
// required arguments and options. Sindri can also be
// safely created directly so long as the exported
// fields are set to non-nil values.
func New(steamAppID string, bepInEx *thunderstore.Package, thunderstoreClient *thunderstore.Client, opts ...Opt) (*Sindri, error) {
	s := &Sindri{
		SteamAppID:         steamAppID,
		BepInEx:            bepInEx,
		ThunderstoreClient: thunderstoreClient,
	}

	return s, s.init(opts...)
}

// Packages returns the installed thunderstore.io packages.
func (s *Sindri) Packages() ([]thunderstore.Package, error) {
	pkgs := []thunderstore.Package{}

	for k, v := range s.metadata.Mods {
		pkg, err := thunderstore.ParsePackage(k + "-" + v.Version)
		if err != nil {
			return nil, err
		}

		pkgs = append(pkgs, *pkg)
	}

	return pkgs, nil
}

// AppUpdate uses `steamcmd` to installed or update
// the game that Sindri is managing.
func (s *Sindri) AppUpdate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp(s.stateDir, "base-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cmd, err := steamcmd.Run(ctx, &steamcmd.Commands{
		ForceInstallDir: tmp,
		AppUpdate:       s.SteamAppID,
		Beta:            s.beta,
		BetaPassword:    s.betaPassword,
		Validate:        true,
	})
	if err != nil {
		return err
	}

	if err = cmd.Run(); err != nil {
		return err
	}

	layer, err := xcontainerregistry.LayerFromDir(tmp)
	if err != nil {
		return err
	}

	digest, err := layer.Digest()
	if err != nil {
		return err
	}

	if s.metadata.BaseLayerDigest == digest.String() {
		return nil
	}

	layers, err := s.modLayers()
	if err != nil {
		return err
	}

	layers = append(layers, layer)

	if s.img, err = mutate.AppendLayers(empty.Image, layers...); err != nil {
		return err
	}

	s.metadata.BaseLayerDigest = digest.String()

	return s.save()
}

// AddMods installs or updates the given mods and their
// dependencies using thunderstore.io.
func (s *Sindri) AddMods(ctx context.Context, mods ...string) ([]thunderstore.Package, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return nil, err
	}

	return s.addMods(ctx, mods...)
}

// Extract returns an io.ReadCloser containing a tarball
// containing the files of the game and its mods.
func (s *Sindri) Extract() (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return mutate.Extract(s.img), nil
}

// ExtractMods returns an io.ReadCloser containing a tarball
// containing the files just the game's mods.
func (s *Sindri) ExtractMods() (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	layers, err := s.modLayers()
	if err != nil {
		return nil, err
	}

	img, err := mutate.AppendLayers(empty.Image, layers...)
	if err != nil {
		return nil, err
	}

	return mutate.Extract(img), nil
}

func (s *Sindri) save() error {
	var (
		tmpTarPath = filepath.Join(s.rootDir, "sindri.tmp.tar")
		tmpDbPath  = filepath.Join(s.rootDir, "sindri.tmp.json")
	)

	if err := tarball.WriteToFile(tmpTarPath, name.MustParseReference(ImageRef), s.img); err != nil {
		return err
	}

	if err := os.Rename(tmpTarPath, s.tarPath()); err != nil {
		return err
	}

	img, err := tarball.ImageFromPath(s.tarPath(), s.tag)
	if err != nil {
		return err
	}

	s.img = img

	db, err := os.Create(tmpDbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err = json.NewEncoder(db).Encode(s.metadata); err != nil {
		return err
	}

	if err := os.Rename(tmpDbPath, s.dbPath()); err != nil {
		return err
	}

	return nil
}

func (s *Sindri) dbPath() string {
	return filepath.Join(s.rootDir, "sindri.json")
}

func (s *Sindri) tarPath() string {
	return filepath.Join(s.rootDir, "sindri.tar")
}

func (s *Sindri) filteredLayers(filter func(v1.Layer) (bool, error)) ([]v1.Layer, error) {
	layers, err := s.img.Layers()
	if err != nil {
		return nil, err
	}

	filteredLayers := []v1.Layer{}

	for _, l := range layers {
		if pass, err := filter(l); err != nil {
			return nil, err
		} else if pass {
			filteredLayers = append(filteredLayers, l)
		}
	}

	return filteredLayers, nil
}

func (s *Sindri) modLayers() ([]v1.Layer, error) {
	return s.filteredLayers(func(l v1.Layer) (bool, error) {
		digest, err := l.Digest()
		if err != nil {
			return false, err
		}

		return digest.String() != s.metadata.BaseLayerDigest, nil
	})
}

func (s *Sindri) addMods(ctx context.Context, mods ...string) ([]thunderstore.Package, error) {
	pkgs := []thunderstore.Package{}

	for _, mod := range mods {
		pkg, err := thunderstore.ParsePackage(mod)
		if err != nil {
			return nil, err
		}

		// The pkg doesn't need a version to get the metadata
		// or the archive, but we want the version so we know
		// what version is installed, so we make sure that we
		// have it. We also need to know its dependencies.
		meta, err := s.ThunderstoreClient.GetPackageMetadata(ctx, pkg)
		if err != nil {
			return nil, err
		}

		if pkg.Version == "" && meta.Latest != nil {
			pkg = &meta.Latest.Package
		}

		versionlessStr := pkg.Versionless().String()

		current, ok := s.metadata.Mods[versionlessStr]
		if ok {
			if current.Version == pkg.Version {
				pkgs = append(pkgs, *pkg)
				continue
			}
		}

		isBepInEx := versionlessStr == s.BepInEx.Versionless().String()

		tmp, err := os.MkdirTemp(s.stateDir, pkg.Fullname()+"-*")
		if err != nil {
			return nil, err
		}

		// Every mod except BepInEx itself is dependent on BepInEx because
		// we use BepInEx to make Valheim load the mod.
		dependencies := meta.Dependencies
		if !(isBepInEx || fn.Some(dependencies, func(dep string, _ int) bool {
			return strings.HasPrefix(dep, s.BepInEx.Versionless().String())
		})) {
			dependencies = append(dependencies, s.BepInEx.Fullname())
		}

		if _, err := s.addMods(ctx, dependencies...); err != nil {
			return nil, err
		}

		zrc, err := s.ThunderstoreClient.GetPackageZip(ctx, pkg)
		if err != nil {
			return nil, err
		}
		defer zrc.Close()

		zr, err := zip.NewReader(zrc, zrc.Size())
		if err != nil {
			return nil, err
		}

		zr.File = fn.Reduce(zr.File, func(acc []*zip.File, cur *zip.File, _ int) []*zip.File {
			norm := strings.ReplaceAll(cur.Name, "\\", "/")

			if isBepInEx {
				name, err := filepath.Rel(s.BepInEx.Name, norm)
				if err != nil {
					return acc
				}

				if strings.Contains(name, "..") {
					return acc
				}

				cur.Name = name
			} else {
				cur.Name = filepath.Join("BepInEx/plugins", pkg.Fullname(), norm)
			}

			return append(acc, cur)
		}, []*zip.File{})

		if err := xzip.Extract(zr, tmp); err != nil {
			return nil, err
		}

		layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
			return xtar.Compress(tmp), nil
		})
		if err != nil {
			return nil, err
		}

		digest, err := layer.Digest()
		if err != nil {
			return nil, err
		}

		if ok && current.LayerDigest == digest.String() {
			pkgs = append(pkgs, *pkg)
			continue
		}

		layers, err := s.filteredLayers(func(l v1.Layer) (bool, error) {
			digest, err := l.Digest()
			if err != nil {
				return false, err
			}

			return !ok || digest.String() != current.LayerDigest, nil
		})
		if err != nil {
			return nil, err
		}

		layers = append(layers, layer)

		if s.img, err = mutate.AppendLayers(s.img, layers...); err != nil {
			return nil, err
		}

		s.metadata.Mods[versionlessStr] = ModMetadata{
			Version:     pkg.Version,
			LayerDigest: digest.String(),
		}
		defer os.RemoveAll(tmp)

		pkgs = append(pkgs, *pkg)
	}

	return pkgs, s.save()
}

func (s *Sindri) init(opts ...Opt) error {
	switch {
	case s.SteamAppID == "":
		return fmt.Errorf("empty SteamAppID")
	case s.BepInEx == nil:
		return fmt.Errorf("nil BepInEx Package")
	case s.ThunderstoreClient == nil:
		return fmt.Errorf("nil ThunderstoreClient")
	}

	if s.initialized {
		return nil
	}

	s.img = empty.Image
	s.mu = new(sync.Mutex)
	s.metadata = &Metadata{
		Mods: map[string]ModMetadata{},
	}

	for _, opt := range opts {
		opt(s)
	}

	tag, err := name.NewTag(ImageRef)
	if err != nil {
		return err
	}
	s.tag = &tag

	if s.rootDir == "" || s.stateDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		if s.rootDir == "" {
			s.rootDir = filepath.Join(wd, "root")
		}

		if s.stateDir == "" {
			s.stateDir = filepath.Join(wd, "state")
		}
	}

	if err := os.MkdirAll(s.stateDir, 0600); err != nil {
		return err
	}

	if err := os.MkdirAll(s.rootDir, 0600); err != nil {
		return err
	}

	if fi, err := os.Stat(s.tarPath()); err == nil && !fi.IsDir() && fi.Size() > 0 {
		if s.img, err = tarball.ImageFromPath(s.tarPath(), s.tag); err != nil {
			return err
		}
	}

	if fi, err := os.Stat(s.dbPath()); err == nil && !fi.IsDir() && fi.Size() > 0 {
		db, err := os.Open(s.dbPath())
		if err != nil {
			return err
		}
		defer db.Close()

		if err = json.NewDecoder(db).Decode(s.metadata); err != nil {
			return err
		}
	}

	s.initialized = true

	return nil
}
