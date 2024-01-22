package sindri

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/thunderstore"
	xtar "github.com/frantjc/x/archive/tar"
	xzip "github.com/frantjc/x/archive/zip"
	xio "github.com/frantjc/x/io"
	xslice "github.com/frantjc/x/slice"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"golang.org/x/exp/maps"
)

// ModMetadata stores metadata about an added mod.
type ModMetadata struct {
	LayerDigest string `json:"layerDigest,omitempty"`
	Version     string `json:"version,omitempty"`
}

// Metadata stores metadata about a downloaded game
// and added mods.
type Metadata struct {
	SteamAppLayerDigest string                 `json:"steamAppLayerDigest,omitempty"`
	Mods                map[string]ModMetadata `json:"mods,omitempty"`
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
	// MetadataLayerDigestLabel is the image config file label
	// that Sindri stores Metadata at.
	MetadataLayerDigestLabel = "cc.frantj.sindri.metadata-layer-digest"
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

// Mods returns the installed thunderstore.io packages.
func (s *Sindri) Mods() ([]thunderstore.Package, error) {
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

type readCloser struct {
	io.Reader
	io.Closer
}

func reproducibleBuildLayerFromDir(dir string) (v1.Layer, error) {
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		var (
			rc1 = xtar.Compress(dir)
			rc2 = xtar.ModifyHeaders(tar.NewReader(rc1), func(h *tar.Header) {
				h.ModTime = sourceDateEpoch
			})
		)

		return &readCloser{
			Reader: rc2,
			Closer: xio.CloserFunc(func() error {
				return errors.Join(rc2.Close(), rc1.Close())
			}),
		}, nil
	})
}

// AppUpdate uses `steamcmd` to installed or update
// the game that *Sindri is managing.
func (s *Sindri) AppUpdate(ctx context.Context) error {
	if s.SteamAppID == "" {
		return fmt.Errorf("empty SteamAppID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return err
	}

	steamcmdForceInstallDir, err := os.MkdirTemp(s.stateDir, "steamapp-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(steamcmdForceInstallDir)

	if err := steamcmd.Command("steamcmd").AppUpdate(ctx, &steamcmd.AppUpdateCombined{
		ForceInstallDir: steamcmdForceInstallDir,
		AppUpdate: &steamcmd.AppUpdate{
			AppID:        s.SteamAppID,
			Beta:         s.beta,
			BetaPassword: s.betaPassword,
			Validate:     true,
		},
	}); err != nil {
		return err
	}

	steamAppLayer, err := reproducibleBuildLayerFromDir(steamcmdForceInstallDir)
	if err != nil {
		return err
	}

	steamAppLayerDigest, err := steamAppLayer.Digest()
	if err != nil {
		return err
	}

	// If the digest hasn't changed, we don't need to spend
	// any more time on this.
	if s.metadata.SteamAppLayerDigest == steamAppLayerDigest.String() {
		return nil
	}

	layers, err := s.img.Layers()
	if err != nil {
		return err
	}

	filteredLayers := []v1.Layer{steamAppLayer}

	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return err
		}

		if s.metadata.SteamAppLayerDigest != digest.String() {
			filteredLayers = append(filteredLayers, layer)
		}
	}

	if s.img, err = mutate.AppendLayers(empty.Image, filteredLayers...); err != nil {
		return err
	}

	s.metadata.SteamAppLayerDigest = steamAppLayerDigest.String()

	return s.save()
}

// AddMods installs or updates the given mods and their
// dependencies using thunderstore.io.
func (s *Sindri) AddMods(ctx context.Context, mods ...string) error {
	switch {
	case len(mods) == 0:
		return nil
	case s.BepInEx == nil:
		return fmt.Errorf("nil BepInEx Package")
	case s.ThunderstoreClient == nil:
		return fmt.Errorf("nil ThunderstoreClient")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return err
	}

	layers, err := s.img.Layers()
	if err != nil {
		return err
	}

	for _, mod := range append(
		xslice.Unique(mods),
		s.BepInEx.Versionless().String(),
	) {
		pkg, err := thunderstore.ParsePackage(mod)
		if err != nil {
			return err
		}

		var (
			modKey      = pkg.Versionless().String()
			modMeta, ok = s.metadata.Mods[modKey]
		)
		// If the mod version hasn't changed, no need to
		// spend any time on it.
		if ok {
			if modMeta.Version == pkg.Version {
				continue
			}
		}

		pkgUnzipDir, err := os.MkdirTemp(s.stateDir, pkg.Fullname()+"-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(pkgUnzipDir)

		if err := s.extractModsAndDependenciesToDir(ctx, pkgUnzipDir, mod); err != nil {
			return err
		}

		modLayer, err := reproducibleBuildLayerFromDir(pkgUnzipDir)
		if err != nil {
			return err
		}

		modLayerDigest, err := modLayer.Digest()
		if err != nil {
			return err
		}

		// If the digest hasn't changed, we don't need to spend
		// any more time on this.
		if ok && modMeta.LayerDigest == modLayerDigest.String() {
			continue
		}

		fileteredLayers := []v1.Layer{}

		for _, layer := range layers {
			digest, err := layer.Digest()
			if err != nil {
				return err
			}

			if digest.String() != modMeta.LayerDigest {
				fileteredLayers = append(fileteredLayers, layer)
			}
		}

		layers = fileteredLayers
		layers = append(layers, modLayer)

		s.metadata.Mods[modKey] = ModMetadata{
			Version:     pkg.Version,
			LayerDigest: modLayerDigest.String(),
		}
	}

	if s.img, err = mutate.AppendLayers(empty.Image, layers...); err != nil {
		return err
	}

	return s.save()
}

// RemoveMods removes the given mods.
func (s *Sindri) RemoveMods(_ context.Context, mods ...string) error {
	if len(mods) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return err
	}

	layers, err := s.img.Layers()
	if err != nil {
		return err
	}

	for _, mod := range mods {
		pkg, err := thunderstore.ParsePackage(mod)
		if err != nil {
			return err
		}

		if pkg.Versionless().String() == s.BepInEx.Versionless().String() {
			return fmt.Errorf("cannot remove BepInEx")
		}

		var (
			modKey      = pkg.Versionless().String()
			modMeta, ok = s.metadata.Mods[modKey]
		)
		if !ok {
			continue
		}

		fileteredLayers := []v1.Layer{}

		for _, layer := range layers {
			digest, err := layer.Digest()
			if err != nil {
				return err
			}

			if digest.String() != modMeta.LayerDigest {
				fileteredLayers = append(fileteredLayers, layer)
			}
		}

		delete(s.metadata.Mods, modKey)
		layers = fileteredLayers
	}

	if s.img, err = mutate.AppendLayers(empty.Image, layers...); err != nil {
		return err
	}

	return s.save()
}

var sourceDateEpoch = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).
	Add(func() time.Duration {
		sourceDateEpoch := os.Getenv("SOURCE_DATE_EPOCH")
		if sourceDateEpoch == "" {
			return 0
		}

		if seconds, err := strconv.Atoi(sourceDateEpoch); err == nil {
			return time.Second * time.Duration(seconds)
		}

		return 0
	}())

// Extract returns an io.ReadCloser of a tarball
// containing the files of the game and the given mods.
func (s *Sindri) Extract(mods ...string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	modLayers, err := s.modLayers(mods...)
	if err != nil {
		return nil, err
	}

	layers, err := s.layerDigests(s.metadata.SteamAppLayerDigest)
	if err != nil {
		return nil, err
	}

	layers = append(layers, modLayers...)

	img, err := mutate.AppendLayers(empty.Image, layers...)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return xtar.ModifyHeaders(tar.NewReader(mutate.Extract(img)), func(h *tar.Header) {
		h.ModTime = now
	}), nil
}

// ExtractMods returns an io.ReadCloser containing a tarball
// containing the files just the game's mods.
func (s *Sindri) ExtractMods(mods ...string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	modLayers, err := s.modLayers(mods...)
	if err != nil {
		return nil, err
	}

	img, err := mutate.AppendLayers(empty.Image, modLayers...)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return xtar.ModifyHeaders(tar.NewReader(mutate.Extract(img)), func(h *tar.Header) {
		h.ModTime = now
	}), nil
}

func (s *Sindri) save() error {
	metadataLayer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		var (
			buf = new(bytes.Buffer)
			tw  = tar.NewWriter(buf)
		)

		b, err := json.Marshal(s.metadata)
		if err != nil {
			return nil, err
		}

		if err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     s.metadataName(),
			Size:     int64(len(b)),
			Mode:     0644,
			ModTime:  sourceDateEpoch,
		}); err != nil {
			return nil, err
		}

		if _, err = tw.Write(b); err != nil {
			return nil, err
		}

		if err = tw.Close(); err != nil {
			return nil, err
		}

		return io.NopCloser(buf), nil
	})
	if err != nil {
		return err
	}

	metadataLayerDigest, err := metadataLayer.Digest()
	if err != nil {
		return err
	}

	configFile, err := s.img.ConfigFile()
	if err != nil {
		configFile = &v1.ConfigFile{
			Config: v1.Config{
				Labels: make(map[string]string),
			},
		}
	}

	oldMetadataLayerDigest := configFile.Config.Labels[MetadataLayerDigestLabel]

	layers, err := s.img.Layers()
	if err != nil {
		return err
	}

	newLayers := []v1.Layer{metadataLayer}

	if oldMetadataLayerDigest != "" {
		for _, layer := range layers {
			digest, err := layer.Digest()
			if err != nil {
				return err
			}

			if digest.String() != oldMetadataLayerDigest {
				newLayers = append(newLayers, layer)
			}
		}
	} else {
		newLayers = append(newLayers, layers...)
	}

	if s.img, err = mutate.AppendLayers(empty.Image, newLayers...); err != nil {
		return err
	}

	configFile, err = s.img.ConfigFile()
	if err != nil {
		configFile = &v1.ConfigFile{
			Config: v1.Config{
				Labels: make(map[string]string),
			},
		}
	} else if configFile.Config.Labels == nil {
		configFile.Config.Labels = map[string]string{}
	}

	maps.Copy(configFile.Config.Labels, map[string]string{
		MetadataLayerDigestLabel: metadataLayerDigest.String(),
	})

	if s.img, err = mutate.ConfigFile(s.img, configFile); err != nil {
		return err
	}

	if err := tarball.WriteToFile(s.tmpDbPath(), name.MustParseReference(ImageRef), s.img); err != nil {
		return err
	}

	if err = os.Rename(s.tmpDbPath(), s.dbPath()); err != nil {
		return err
	}

	img, err := tarball.ImageFromPath(s.dbPath(), s.tag)
	if err != nil {
		return err
	}

	s.img = img

	return nil
}

func (s *Sindri) extractModsAndDependenciesToDir(ctx context.Context, dir string, mods ...string) error {
	errC := make(chan error, 1)

	for _, m := range mods {
		go func(mod string) {
			pkg, err := thunderstore.ParsePackage(mod)
			if err != nil {
				errC <- err
				return
			}

			var (
				modKey      = pkg.Versionless().String()
				modMeta, ok = s.metadata.Mods[modKey]
			)
			if ok {
				if modMeta.Version == pkg.Version {
					return
				}
			}

			// The pkg doesn't need a version to get the metadata
			// or the archive, but we want the version so we know
			// what version is installed, so we make sure that we
			// have it. We also need to know its dependencies.
			pkgMeta, err := s.ThunderstoreClient.GetPackageMetadata(ctx, pkg)
			if err != nil {
				errC <- err
				return
			}

			var (
				bepInExKey = s.BepInEx.Versionless().String()
				isBepInEx  = modKey == bepInExKey
			)

			dependencies := pkgMeta.Dependencies

			if pkg.Version == "" && pkgMeta.Latest != nil {
				pkg = &pkgMeta.Latest.Package
				dependencies = append(dependencies, pkgMeta.Latest.Dependencies...)
			}

			dependencies = xslice.Unique(xslice.Filter(dependencies, func(dependency string, _ int) bool {
				return !strings.HasPrefix(dependency, bepInExKey)
			}))

			if err := s.extractModsAndDependenciesToDir(ctx, dir, dependencies...); err != nil {
				errC <- err
				return
			}

			pkgZip, err := s.ThunderstoreClient.GetPackageZip(ctx, pkg)
			if err != nil {
				errC <- err
				return
			}
			defer pkgZip.Close()

			pkgZipRdr, err := zip.NewReader(pkgZip, pkgZip.Size())
			if err != nil {
				errC <- err
				return
			}

			pkgZipRdr.File = xslice.Reduce(pkgZipRdr.File, func(acc []*zip.File, cur *zip.File, _ int) []*zip.File {
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

			if err := xzip.Extract(pkgZipRdr, dir); err != nil {
				errC <- err
				return
			}

			errC <- nil
		}(m)
	}

	for i := 0; i < len(mods); i++ {
		if err := <-errC; err != nil {
			return err
		}
	}

	return nil
}

func (s *Sindri) dbPath() string {
	return filepath.Join(s.rootDir, "sindri.db")
}

func (s *Sindri) tmpDbPath() string {
	return filepath.Join(s.rootDir, "sindri.tmp.db")
}

func (s *Sindri) metadataName() string {
	return "sindri.metadata.json"
}

func (s *Sindri) modLayers(mods ...string) ([]v1.Layer, error) {
	if len(mods) == 0 {
		return []v1.Layer{}, nil
	}

	mods = xslice.Unique(append(mods, s.BepInEx.Versionless().String()))

	var (
		extractLayerDigests = []string{}
		lenMods             = len(mods)
	)

	for _, mod := range mods {
		pkg, err := thunderstore.ParsePackage(mod)
		if err != nil {
			return nil, err
		}

		if modMeta, ok := s.metadata.Mods[pkg.Versionless().String()]; ok {
			extractLayerDigests = append(extractLayerDigests, modMeta.LayerDigest)

			if len(extractLayerDigests) == lenMods {
				// Found them all
				break
			}
		} else {
			return nil, fmt.Errorf("couldn't find mod " + mod + " layer digest")
		}
	}

	if len(extractLayerDigests) != lenMods {
		return nil, fmt.Errorf("unable to find all mod layer digests")
	}

	return s.layerDigests(extractLayerDigests...)
}

func (s *Sindri) layerDigests(layerDigests ...string) ([]v1.Layer, error) {
	layers, err := s.img.Layers()
	if err != nil {
		return nil, err
	}

	var (
		filteredLayers  = []v1.Layer{}
		lenLayerDigests = len(layerDigests)
	)

	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return nil, err
		}

		if xslice.Includes(layerDigests, digest.String()) {
			filteredLayers = append(filteredLayers, layer)

			if len(filteredLayers) == lenLayerDigests {
				// Found them all
				break
			}
		}
	}

	if len(filteredLayers) != lenLayerDigests {
		return nil, fmt.Errorf("unable to find all layers by digest")
	}

	return filteredLayers, nil
}

func (s *Sindri) init(opts ...Opt) error {
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

	if err := os.MkdirAll(s.stateDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(s.rootDir, 0755); err != nil {
		return err
	}

	if fi, err := os.Stat(s.dbPath()); err == nil && !fi.IsDir() && fi.Size() > 0 {
		if s.img, err = tarball.ImageFromPath(s.dbPath(), s.tag); err != nil {
			return err
		}
	}

	configFile, err := s.img.ConfigFile()
	if err != nil {
		return err
	}

	if metadataLayerDigest, ok := configFile.Config.Labels[MetadataLayerDigestLabel]; ok {
		layers, err := s.img.Layers()
		if err != nil {
			return err
		}

		var (
			found        = false
			metadataName = s.metadataName()
		)

		for _, layer := range layers {
			digest, err := layer.Digest()
			if err != nil {
				return err
			}

			if digest.String() == metadataLayerDigest {
				found = true

				rc, err := layer.Uncompressed()
				if err != nil {
					return err
				}
				defer rc.Close()

				metadataTarReader := tar.NewReader(rc)
				for {
					hdr, err := metadataTarReader.Next()
					if errors.Is(err, io.EOF) {
						return fmt.Errorf("unable to find metadata in metadata layer")
					} else if err != nil {
						return err
					}

					if hdr.Name == metadataName {
						if err = json.NewDecoder(metadataTarReader).Decode(s.metadata); err != nil {
							return err
						}

						break
					}
				}
			}
		}

		if !found {
			return fmt.Errorf("unable to find metadata layer")
		}
	}

	s.initialized = true

	return nil
}
