package gui

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// thumbnailSize is the list-row display size; thumbnails are decoded at
// thumbnailDecodeSize so grid cells and previews are not upscaled.
const (
	thumbnailSize       = 96
	thumbnailDecodeSize = 240
)

// thumbnailCache decodes and downscales chapter images once per path.
type thumbnailCache struct {
	mu    sync.Mutex
	items map[string]image.Image
}

func newThumbnailCache() *thumbnailCache {
	return &thumbnailCache{items: map[string]image.Image{}}
}

// Get returns a thumbnail for path, or nil when the image cannot be decoded.
func (c *thumbnailCache) Get(path string) image.Image {
	c.mu.Lock()
	if img, ok := c.items[path]; ok {
		c.mu.Unlock()
		return img
	}
	c.mu.Unlock()

	img := loadThumbnail(path, thumbnailDecodeSize)

	c.mu.Lock()
	c.items[path] = img
	c.mu.Unlock()
	return img
}

// fill sets img from the cache immediately when possible; otherwise it decodes
// in the background and calls onReady on the UI thread once done.
func (c *thumbnailCache) fill(img *canvas.Image, path string, onReady func()) {
	c.mu.Lock()
	cached, ok := c.items[path]
	c.mu.Unlock()
	if ok {
		img.Image = cached
		img.Refresh()
		return
	}
	img.Image = nil
	img.Refresh()
	go func() {
		c.Get(path)
		fyne.Do(onReady)
	}()
}

func loadThumbnail(path string, maxEdge int) image.Image {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return scaleToFit(src, maxEdge)
}

func scaleToFit(src image.Image, maxEdge int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return nil
	}
	scale := float64(maxEdge) / float64(max(w, h))
	if scale >= 1 {
		return src
	}
	dw, dh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}
