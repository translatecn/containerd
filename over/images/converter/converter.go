// Package converter provides image converter
package converter

import (
	"context"
	"demo/over/content"
	"demo/over/images"
	"demo/over/leases"
	"demo/over/platforms"
)

type convertOpts struct {
	layerConvertFunc ConvertFunc
	docker2oci       bool
	indexConvertFunc ConvertFunc
	platformMC       platforms.MatchComparer
}

// Opt is an option for Convert()
type Opt func(*convertOpts) error

// WithLayerConvertFunc specifies the function that converts layers.
func WithLayerConvertFunc(fn ConvertFunc) Opt {
	return func(copts *convertOpts) error {
		copts.layerConvertFunc = fn
		return nil
	}
}

// WithDockerToOCI converts Docker media types into OCI ones.
func WithDockerToOCI(v bool) Opt {
	return func(copts *convertOpts) error {
		copts.docker2oci = true
		return nil
	}
}

// WithPlatform specifies the platform.
// Defaults to all platforms.
func WithPlatform(p platforms.MatchComparer) Opt {
	return func(copts *convertOpts) error {
		copts.platformMC = p
		return nil
	}
}

// WithIndexConvertFunc specifies the function that converts manifests and index (manifest lists).
// Defaults to DefaultIndexConvertFunc.

// Client is implemented by *containerd.Client .
type Client interface {
	WithLease(ctx context.Context, opts ...leases.Opt) (context.Context, func(context.Context) error, error)
	ContentStore() content.Store
	ImageService() images.Store
}

// Convert converts an image.
func Convert(ctx context.Context, client Client, dstRef, srcRef string, opts ...Opt) (*images.Image, error) {
	var copts convertOpts
	for _, o := range opts {
		if err := o(&copts); err != nil {
			return nil, err
		}
	}
	if copts.platformMC == nil {
		copts.platformMC = platforms.All
	}
	if copts.indexConvertFunc == nil {
		copts.indexConvertFunc = DefaultIndexConvertFunc(copts.layerConvertFunc, copts.docker2oci, copts.platformMC)
	}

	ctx, done, err := client.WithLease(ctx)
	if err != nil {
		return nil, err
	}
	defer done(ctx)

	cs := client.ContentStore()
	is := client.ImageService()
	srcImg, err := is.Get(ctx, srcRef)
	if err != nil {
		return nil, err
	}

	dstDesc, err := copts.indexConvertFunc(ctx, cs, srcImg.Target)
	if err != nil {
		return nil, err
	}

	dstImg := srcImg
	dstImg.Name = dstRef
	if dstDesc != nil {
		dstImg.Target = *dstDesc
	}
	var res images.Image
	if dstRef != srcRef {
		_ = is.Delete(ctx, dstRef)
		res, err = is.Create(ctx, dstImg)
	} else {
		res, err = is.Update(ctx, dstImg)
	}
	return &res, err
}
