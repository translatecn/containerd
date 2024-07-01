package sbserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	criconfig "demo/config/cri"
	tracing "demo/pkg/ctr_tracing"
	"demo/pkg/imgcrypt"
	"demo/pkg/imgcrypt/images/encryption"
	"demo/pkg/log"
	distribution "demo/pkg/reference/docker"
	snpkg "demo/pkg/snapshotters"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	runtime "demo/pkg/api/cri/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"

	"demo/pkg/containerd"
	"demo/pkg/cri/annotations"
	crilabels "demo/pkg/cri/labels"
	"demo/pkg/errdefs"
	containerdimages "demo/pkg/images"
	"demo/pkg/remotes/docker"
	"demo/pkg/remotes/docker/config"
)

// createImageReference creates image reference inside containerd image store.
// Note that because create and update are not finished in one transaction, there could be race. E.g.
// the image reference is deleted by someone else after create returns already exists, but before update
// happens.
func (c *CriService) createImageReference(ctx context.Context, name string, desc imagespec.Descriptor, labels map[string]string) error {
	img := containerdimages.Image{
		Name:   name,
		Target: desc,
		// Add a label to indicate that the image is managed by the cri plugin.
		Labels: labels,
	}
	// TODO(random-liu): Figure out which is the more performant sequence create then update or
	// update then create.
	_, err := c.client.ImageService().Create(ctx, img)
	if err == nil || !errdefs.IsAlreadyExists(err) {
		return err
	}
	// Retrieve oldImg from image store here because Create routine returns an
	// empty image on ErrAlreadyExists
	oldImg, err := c.client.ImageService().Get(ctx, name)
	if err != nil {
		return err
	}
	fieldpaths := []string{"target"}
	if oldImg.Labels[crilabels.ImageLabelKey] != labels[crilabels.ImageLabelKey] {
		fieldpaths = append(fieldpaths, "labels."+crilabels.ImageLabelKey)
	}
	if oldImg.Labels[crilabels.PinnedImageLabelKey] != labels[crilabels.PinnedImageLabelKey] &&
		labels[crilabels.PinnedImageLabelKey] == crilabels.PinnedImageLabelValue {
		fieldpaths = append(fieldpaths, "labels."+crilabels.PinnedImageLabelKey)
	}
	if oldImg.Target.Digest == img.Target.Digest && len(fieldpaths) < 2 {
		return nil
	}
	_, err = c.client.ImageService().Update(ctx, img, fieldpaths...)
	return err
}

// getLabels get image labels to be added on CRI image
func (c *CriService) getLabels(ctx context.Context, name string) map[string]string {
	labels := map[string]string{crilabels.ImageLabelKey: crilabels.ImageLabelValue}
	configSandboxImage := c.config.SandboxImage
	// parse sandbox image
	sandboxNamedRef, err := distribution.ParseDockerRef(configSandboxImage)
	if err != nil {
		log.G(ctx).Errorf("failed to parse sandbox image from config %s", sandboxNamedRef)
		return nil
	}
	sandboxRef := sandboxNamedRef.String()
	// Adding pinned image label to sandbox image
	if sandboxRef == name {
		labels[crilabels.PinnedImageLabelKey] = crilabels.PinnedImageLabelValue
	}
	return labels
}

// updateImage updates image store to reflect the newest state of an image reference
// in containerd. If the reference is not managed by the cri plugin, the function also
// generates necessary metadata for the image and make it managed.
func (c *CriService) updateImage(ctx context.Context, r string) error {
	img, err := c.client.GetImage(ctx, r)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("get image by reference: %w", err)
	}
	if err == nil && img.Labels()[crilabels.ImageLabelKey] != crilabels.ImageLabelValue {
		// Make sure the image has the image id as its unique
		// identifier that references the image in its lifetime.
		configDesc, err := img.Config(ctx)
		if err != nil {
			return fmt.Errorf("get image id: %w", err)
		}
		id := configDesc.Digest.String()
		labels := c.getLabels(ctx, r)
		if err := c.createImageReference(ctx, id, img.Target(), labels); err != nil {
			return fmt.Errorf("create image id reference %q: %w", id, err)
		}
		if err := c.imageStore.Update(ctx, id); err != nil {
			return fmt.Errorf("update image store for %q: %w", id, err)
		}
		// The image id is ready, add the label to mark the image as managed.
		if err := c.createImageReference(ctx, r, img.Target(), labels); err != nil {
			return fmt.Errorf("create managed label: %w", err)
		}
	}
	// If the image is not found, we should continue updating the cache,
	// so that the image can be removed from the cache.
	if err := c.imageStore.Update(ctx, r); err != nil {
		return fmt.Errorf("update image store for %q: %w", r, err)
	}
	return nil
}

// getTLSConfig returns a TLSConfig configured with a CA/Cert/Key specified by registryTLSConfig
func (c *CriService) getTLSConfig(registryTLSConfig criconfig.TLSConfig) (*tls.Config, error) {
	var (
		tlsConfig = &tls.Config{}
		cert      tls.Certificate
		err       error
	)
	if registryTLSConfig.CertFile != "" && registryTLSConfig.KeyFile == "" {
		return nil, fmt.Errorf("cert file %q was specified, but no corresponding key file was specified", registryTLSConfig.CertFile)
	}
	if registryTLSConfig.CertFile == "" && registryTLSConfig.KeyFile != "" {
		return nil, fmt.Errorf("key file %q was specified, but no corresponding cert file was specified", registryTLSConfig.KeyFile)
	}
	if registryTLSConfig.CertFile != "" && registryTLSConfig.KeyFile != "" {
		cert, err = tls.LoadX509KeyPair(registryTLSConfig.CertFile, registryTLSConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load cert file: %w", err)
		}
		if len(cert.Certificate) != 0 {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		// TODO(thaJeztah): verify if we should ignore the deprecation; see https://github.com/containerd/pull/7349/files#r990644833
		tlsConfig.BuildNameToCertificate() //nolint:staticcheck
	}

	if registryTLSConfig.CAFile != "" {
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to get system cert pool: %w", err)
		}
		caCert, err := os.ReadFile(registryTLSConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA file: %w", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	tlsConfig.InsecureSkipVerify = registryTLSConfig.InsecureSkipVerify
	return tlsConfig, nil
}

func hostDirFromRoots(roots []string) func(string) (string, error) {
	rootfn := make([]func(string) (string, error), len(roots))
	for i := range roots {
		rootfn[i] = config.HostDirFromRoot(roots[i])
	}
	return func(host string) (dir string, err error) {
		for _, fn := range rootfn {
			dir, err = fn(host)
			if (err != nil && !errdefs.IsNotFound(err)) || (dir != "") {
				break
			}
		}
		return
	}
}

// defaultScheme returns the default scheme for a registry host.
func defaultScheme(host string) string {
	if docker.IsLocalhost(host) {
		return "http"
	}
	return "https"
}

// addDefaultScheme returns the endpoint with default scheme
func addDefaultScheme(endpoint string) (string, error) {
	if strings.Contains(endpoint, "://") {
		return endpoint, nil
	}
	ue := "dummy://" + endpoint
	u, err := url.Parse(ue)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s", defaultScheme(u.Host), endpoint), nil
}

// registryEndpoints returns endpoints for a given host.
// It adds default registry endpoint if it does not exist in the passed-in endpoint list.
// It also supports wildcard host matching with `*`.
func (c *CriService) registryEndpoints(host string) ([]string, error) {
	var endpoints []string
	_, ok := c.config.Registry.Mirrors[host]
	if ok {
		endpoints = c.config.Registry.Mirrors[host].Endpoints
	} else {
		endpoints = c.config.Registry.Mirrors["*"].Endpoints
	}
	defaultHost, err := docker.DefaultHost(host)
	if err != nil {
		return nil, fmt.Errorf("get default host: %w", err)
	}
	for i := range endpoints {
		en, err := addDefaultScheme(endpoints[i])
		if err != nil {
			return nil, fmt.Errorf("parse endpoint url: %w", err)
		}
		endpoints[i] = en
	}
	for _, e := range endpoints {
		u, err := url.Parse(e)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint url: %w", err)
		}
		if u.Host == host {
			// Do not add default if the endpoint already exists.
			return endpoints, nil
		}
	}
	return append(endpoints, defaultScheme(defaultHost)+"://"+defaultHost), nil
}

// newTransport returns a new HTTP transport used to pull image.
// TODO(random-liu): Create a library and share this code with `ctr`.
func newTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:       30 * time.Second,
			KeepAlive:     30 * time.Second,
			FallbackDelay: 300 * time.Millisecond,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	}
}

// encryptedImagesPullOpts returns the necessary list of pull options required
// for decryption of encrypted images based on the cri decryption configuration.
func (c *CriService) encryptedImagesPullOpts() []containerd.RemoteOpt {
	if c.config.ImageDecryption.KeyModel == criconfig.KeyModelNode {
		ltdd := imgcrypt.Payload{}
		decUnpackOpt := encryption.WithUnpackConfigApplyOpts(encryption.WithDecryptedUnpack(&ltdd))
		opt := containerd.WithUnpackOpts([]containerd.UnpackOpt{decUnpackOpt})
		return []containerd.RemoteOpt{opt}
	}
	return nil
}

const (
	// minPullProgressReportInternal is used to prevent the reporter from
	// eating more CPU resources
	minPullProgressReportInternal = 5 * time.Second
	// defaultPullProgressReportInterval represents that how often the
	// reporter checks that pull progress.
	defaultPullProgressReportInterval = 10 * time.Second
)

// pullProgressReporter is used to check single PullImage progress.
type pullProgressReporter struct {
	ref         string
	cancel      context.CancelFunc
	reqReporter pullRequestReporter
	timeout     time.Duration
}

func newPullProgressReporter(ref string, cancel context.CancelFunc, timeout time.Duration) *pullProgressReporter {
	return &pullProgressReporter{
		ref:         ref,
		cancel:      cancel,
		reqReporter: pullRequestReporter{},
		timeout:     timeout,
	}
}

func (reporter *pullProgressReporter) optionUpdateClient(client *http.Client) error {
	client.Transport = &pullRequestReporterRoundTripper{
		rt:          client.Transport,
		reqReporter: &reporter.reqReporter,
	}
	return nil
}

func (reporter *pullProgressReporter) start(ctx context.Context) {
	if reporter.timeout == 0 {
		log.G(ctx).Infof("no timeout and will not start pulling image %s reporter", reporter.ref)
		return
	}

	go func() {
		var (
			reportInterval = defaultPullProgressReportInterval

			lastSeenBytesRead = uint64(0)
			lastSeenTimestamp = time.Now()
		)

		// check progress more frequently if timeout < default internal
		if reporter.timeout < reportInterval {
			reportInterval = reporter.timeout / 2

			if reportInterval < minPullProgressReportInternal {
				reportInterval = minPullProgressReportInternal
			}
		}

		var ticker = time.NewTicker(reportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				activeReqs, bytesRead := reporter.reqReporter.status()

				log.G(ctx).WithField("ref", reporter.ref).
					WithField("activeReqs", activeReqs).
					WithField("totalBytesRead", bytesRead).
					WithField("lastSeenBytesRead", lastSeenBytesRead).
					WithField("lastSeenTimestamp", lastSeenTimestamp).
					WithField("reportInterval", reportInterval).
					Tracef("progress for image pull")

				if activeReqs == 0 || bytesRead > lastSeenBytesRead {
					lastSeenBytesRead = bytesRead
					lastSeenTimestamp = time.Now()
					continue
				}

				if time.Since(lastSeenTimestamp) > reporter.timeout {
					log.G(ctx).Errorf("cancel pulling image %s because of no progress in %v", reporter.ref, reporter.timeout)
					reporter.cancel()
					return
				}
			case <-ctx.Done():
				activeReqs, bytesRead := reporter.reqReporter.status()
				log.G(ctx).Infof("stop pulling image %s: active requests=%v, bytes read=%v", reporter.ref, activeReqs, bytesRead)
				return
			}
		}
	}()
}

// countingReadCloser wraps http.Response.Body with pull request reporter,
// which is used by pullRequestReporterRoundTripper.
type countingReadCloser struct {
	once sync.Once

	rc          io.ReadCloser
	reqReporter *pullRequestReporter
}

// Read reads bytes from original io.ReadCloser and increases bytes in
// pull request reporter.
func (r *countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	r.reqReporter.incByteRead(uint64(n))
	return n, err
}

// Close closes the original io.ReadCloser and only decreases the number of
// active pull requests once.
func (r *countingReadCloser) Close() error {
	err := r.rc.Close()
	r.once.Do(r.reqReporter.decRequest)
	return err
}

// pullRequestReporter is used to track the progress per each criapi.PullImage.
type pullRequestReporter struct {
	// activeReqs indicates that current number of active pulling requests,
	// including auth requests.
	activeReqs int32
	// totalBytesRead indicates that the total bytes has been read from
	// remote registry.
	totalBytesRead uint64
}

func (reporter *pullRequestReporter) incRequest() {
	atomic.AddInt32(&reporter.activeReqs, 1)
}

func (reporter *pullRequestReporter) decRequest() {
	atomic.AddInt32(&reporter.activeReqs, -1)
}

func (reporter *pullRequestReporter) incByteRead(nr uint64) {
	atomic.AddUint64(&reporter.totalBytesRead, nr)
}

func (reporter *pullRequestReporter) status() (currentReqs int32, totalBytesRead uint64) {
	currentReqs = atomic.LoadInt32(&reporter.activeReqs)
	totalBytesRead = atomic.LoadUint64(&reporter.totalBytesRead)
	return currentReqs, totalBytesRead
}

// pullRequestReporterRoundTripper wraps http.RoundTripper with pull request
// reporter which is used to track the progress of active http request with
// counting readable http.Response.Body.
//
// NOTE:
//
// Although containerd provides ingester manager to track the progress
// of pulling request, for example `ctr image pull` shows the console progress
// bar, it needs more CPU resources to open/read the ingested files with
// acquiring containerd metadata plugin's boltdb lock.
//
// Before sending HTTP request to registry, the containerd.Client.Pull library
// will open writer by containerd ingester manager. Based on this, the
// http.RoundTripper wrapper can track the active progress with lower overhead
// even if the ref has been locked in ingester manager by other Pull request.
type pullRequestReporterRoundTripper struct {
	rt http.RoundTripper

	reqReporter *pullRequestReporter
}

func (rt *pullRequestReporterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.reqReporter.incRequest()

	resp, err := rt.rt.RoundTrip(req)
	if err != nil {
		rt.reqReporter.decRequest()
		return nil, err
	}

	resp.Body = &countingReadCloser{
		rc:          resp.Body,
		reqReporter: rt.reqReporter,
	}
	return resp, err
}

// -------------------------------------------------------------------------------------------------------------------

// Given that runtime information is not passed from PullImageRequest, we depend on an experimental annotation
// passed from pod sandbox config to get the runtimeHandler. The annotation key is specified in configuration.
// Once we know the runtime, try to override default snapshotter if it is set for this runtime.
// See https://github.com/containerd/issues/6657
func (c *CriService) snapshotterFromPodSandboxConfig(ctx context.Context, imageRef string,
	s *runtime.PodSandboxConfig) (string, error) {
	snapshotter := c.config.ContainerdConfig.Snapshotter
	if s == nil || s.Annotations == nil {
		return snapshotter, nil
	}

	runtimeHandler, ok := s.Annotations[annotations.RuntimeHandler]
	if !ok {
		return snapshotter, nil
	}

	ociRuntime, err := c.getSandboxRuntime(s, runtimeHandler)
	if err != nil {
		return "", fmt.Errorf("experimental: failed to get sandbox runtime for %s, err: %+v", runtimeHandler, err)
	}

	snapshotter = c.runtimeSnapshotter(context.Background(), ociRuntime)
	log.G(ctx).Infof("experimental: PullImage %q for runtime %s, using snapshotter %s", imageRef, runtimeHandler, snapshotter)
	return snapshotter, nil
}
func (c *CriService) PullImage(ctx context.Context, r *runtime.PullImageRequest) (_ *runtime.PullImageResponse, err error) {
	span := tracing.SpanFromContext(ctx)
	defer func() {
		// TODO: add domain label for imagePulls metrics, and we may need to provide a mechanism
		// for the user to configure the set of registries that they are interested in.
		if err != nil {
			imagePulls.WithValues("failure").Inc()
		} else {
			imagePulls.WithValues("success").Inc()
		}
	}()

	inProgressImagePulls.Inc()
	defer inProgressImagePulls.Dec()
	startTime := time.Now()

	imageRef := r.GetImage().GetImage()
	namedRef, err := distribution.ParseDockerRef(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference %q: %w", imageRef, err)
	}
	ref := namedRef.String()
	if ref != imageRef {
		log.G(ctx).Debugf("PullImage using normalized image ref: %q", ref)
	}

	imagePullProgressTimeout, err := time.ParseDuration(c.config.ImagePullProgressTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image_pull_progress_timeout %q: %w", c.config.ImagePullProgressTimeout, err)
	}

	var (
		pctx, pcancel = context.WithCancel(ctx)

		pullReporter = newPullProgressReporter(ref, pcancel, imagePullProgressTimeout)

		resolver = docker.NewResolver(docker.ResolverOptions{
			Headers: c.config.Registry.Headers, // 配置文件里写的
			Hosts:   c.registryHosts(ctx, r.GetAuth(), pullReporter.optionUpdateClient),
		})
		isSchema1    bool
		imageHandler containerdimages.HandlerFunc = func(_ context.Context,
			desc imagespec.Descriptor) ([]imagespec.Descriptor, error) {
			if desc.MediaType == containerdimages.MediaTypeDockerSchema1Manifest {
				isSchema1 = true
			}
			return nil, nil
		}
	)

	defer pcancel()
	snapshotter, err := c.snapshotterFromPodSandboxConfig(ctx, ref, r.SandboxConfig)
	if err != nil {
		return nil, err
	}
	log.G(ctx).Debugf("PullImage %q with snapshotter %s", ref, snapshotter)
	span.SetAttributes(
		tracing.Attribute("image.ref", ref),
		tracing.Attribute("snapshotter.name", snapshotter),
	)

	labels := c.getLabels(ctx, ref)

	pullOpts := []containerd.RemoteOpt{
		containerd.WithSchema1Conversion, //nolint:staticcheck // Ignore SA1019. Need to keep deprecated package for compatibility.
		containerd.WithResolver(resolver),
		containerd.WithPullSnapshotter(snapshotter),
		containerd.WithPullUnpack,
		containerd.WithPullLabels(labels),
		containerd.WithMaxConcurrentDownloads(c.config.MaxConcurrentDownloads),
		containerd.WithImageHandler(imageHandler),
		containerd.WithUnpackOpts([]containerd.UnpackOpt{
			containerd.WithUnpackDuplicationSuppressor(c.unpackDuplicationSuppressor),
		}),
	}

	pullOpts = append(pullOpts, c.encryptedImagesPullOpts()...)
	if !c.config.ContainerdConfig.DisableSnapshotAnnotations {
		pullOpts = append(pullOpts,
			containerd.WithImageHandlerWrapper(snpkg.AppendInfoHandlerWrapper(ref)))
	}

	if c.config.ContainerdConfig.DiscardUnpackedLayers {
		// Allows GC to clean layers up from the content store after unpacking
		pullOpts = append(pullOpts,
			containerd.WithChildLabelMap(containerdimages.ChildGCLabelsFilterLayers))
	}

	pullReporter.start(pctx)
	image, err := c.client.Pull(pctx, ref, pullOpts...)
	pcancel()
	if err != nil {
		return nil, fmt.Errorf("failed to pull and unpack image %q: %w", ref, err)
	}
	span.AddEvent("Pull and unpack image complete")

	configDesc, err := image.Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("get image config descriptor: %w", err)
	}
	imageID := configDesc.Digest.String()

	repoDigest, repoTag := getRepoDigestAndTag(namedRef, image.Target().Digest, isSchema1)
	for _, r := range []string{imageID, repoTag, repoDigest} {
		if r == "" {
			continue
		}
		if err := c.createImageReference(ctx, r, image.Target(), labels); err != nil {
			return nil, fmt.Errorf("failed to create image reference %q: %w", r, err)
		}
		// Update image store to reflect the newest state in containerd.
		// No need to use `updateImage`, because the image reference must
		// have been managed by the cri plugin.
		if err := c.imageStore.Update(ctx, r); err != nil {
			return nil, fmt.Errorf("failed to update image store %q: %w", r, err)
		}
	}

	const mbToByte = 1024 * 1024
	size, _ := image.Size(ctx)
	imagePullingSpeed := float64(size) / mbToByte / time.Since(startTime).Seconds()
	imagePullThroughput.Observe(imagePullingSpeed)

	log.G(ctx).Infof("Pulled image %q with image id %q, repo tag %q, repo digest %q, size %q in %s", imageRef, imageID,
		repoTag, repoDigest, strconv.FormatInt(size, 10), time.Since(startTime))
	// NOTE(random-liu): the actual state in containerd is the source of truth, even we maintain
	// in-memory image store, it's only for in-memory indexing. The image could be removed
	// by someone else anytime, before/during/after we create the metadata. We should always
	// check the actual state in containerd before using the image or returning status of the
	// image.
	return &runtime.PullImageResponse{ImageRef: imageID}, nil
}
func (c *CriService) registryHosts(ctx context.Context, auth *runtime.AuthConfig, updateClientFn config.UpdateClientFunc) docker.RegistryHosts {
	paths := filepath.SplitList(c.config.Registry.ConfigPath)
	if len(paths) > 0 {
		hostOptions := config.HostOptions{
			UpdateClient: updateClientFn,
		}
		hostOptions.Credentials = func(host string) (string, string, error) {
			hostauth := auth
			if hostauth == nil {
				config := c.config.Registry.Configs[host]
				if config.Auth != nil {
					hostauth = toRuntimeAuthConfig(*config.Auth)
				}
			}
			return ParseAuth(hostauth, host)
		}
		hostOptions.HostDir = hostDirFromRoots(paths)

		return config.ConfigureHosts(ctx, hostOptions)
	}

	return func(host string) ([]docker.RegistryHost, error) {
		var registries []docker.RegistryHost

		endpoints, err := c.registryEndpoints(host)
		if err != nil {
			return nil, fmt.Errorf("get registry endpoints: %w", err)
		}
		for _, e := range endpoints {
			u, err := url.Parse(e)
			if err != nil {
				return nil, fmt.Errorf("parse registry endpoint %q from mirrors: %w", e, err)
			}

			var (
				transport = newTransport()
				client    = &http.Client{Transport: transport}
				config    = c.config.Registry.Configs[u.Host]
			)

			if config.TLS != nil {
				transport.TLSClientConfig, err = c.getTLSConfig(*config.TLS)
				if err != nil {
					return nil, fmt.Errorf("get TLSConfig for registry %q: %w", e, err)
				}
			} else if docker.IsLocalhost(host) && u.Scheme == "http" {
				// Skipping TLS verification for localhost
				transport.TLSClientConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
			}

			// Make a copy of `auth`, so that different authorizers would not reference
			// the same auth variable.
			auth := auth
			if auth == nil && config.Auth != nil {
				auth = toRuntimeAuthConfig(*config.Auth)
			}

			if updateClientFn != nil {
				if err := updateClientFn(client); err != nil {
					return nil, fmt.Errorf("failed to update http client: %w", err)
				}
			}

			authorizer := docker.NewDockerAuthorizer(
				docker.WithAuthClient(client),
				docker.WithAuthCreds(func(host string) (string, string, error) {
					return ParseAuth(auth, host)
				}))

			if u.Path == "" {
				u.Path = "/v2"
			}

			registries = append(registries, docker.RegistryHost{
				Client:       client,
				Authorizer:   authorizer,
				Host:         u.Host,
				Scheme:       u.Scheme,
				Path:         u.Path,
				Capabilities: docker.HostCapabilityResolve | docker.HostCapabilityPull,
			})
		}
		return registries, nil
	}
}

// ParseAuth parses AuthConfig and returns username and password/secret required by containerd.
func ParseAuth(auth *runtime.AuthConfig, host string) (string, string, error) {
	if auth == nil {
		return "", "", nil
	}
	if auth.ServerAddress != "" {
		// Do not return the auth info when server address doesn't match.
		u, err := url.Parse(auth.ServerAddress)
		if err != nil {
			return "", "", fmt.Errorf("parse server address: %w", err)
		}
		if host != u.Host {
			return "", "", nil
		}
	}
	if auth.Username != "" {
		return auth.Username, auth.Password, nil
	}
	if auth.IdentityToken != "" {
		return "", auth.IdentityToken, nil
	}
	if auth.Auth != "" {
		decLen := base64.StdEncoding.DecodedLen(len(auth.Auth))
		decoded := make([]byte, decLen)
		_, err := base64.StdEncoding.Decode(decoded, []byte(auth.Auth))
		if err != nil {
			return "", "", err
		}
		user, passwd, ok := strings.Cut(string(decoded), ":")
		if !ok {
			return "", "", fmt.Errorf("invalid decoded auth: %q", decoded)
		}
		return user, strings.Trim(passwd, "\x00"), nil
	}
	// TODO(random-liu): Support RegistryToken.
	// An empty auth config is valid for anonymous registry
	return "", "", nil
}
