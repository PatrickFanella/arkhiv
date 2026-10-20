package sourcecatalog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// Artist photographs are stored as engine-owned renditions so pages never
// load images from the provider. Renditions are re-encoded, which drops
// embedded metadata such as camera location, and are addressed by SHA-256.
const (
	ArtistDisplayWidth   = 1280
	ArtistThumbnailWidth = 330
	maxArtistImageBytes  = 12 << 20
	maxArtistImagePixels = 40_000_000
)

var mediaSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)

type ArtistMedia struct {
	SHA256      string `json:"sha256"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	data        []byte
}

func ValidMediaID(id string) bool { return mediaSHA256.MatchString(id) }

// artistRenditions decodes a retained photograph and produces the bounded
// display image and the list thumbnail. Images with transparency stay PNG.
func artistRenditions(raw []byte) (display, thumbnail ArtistMedia, err error) {
	if len(raw) == 0 || len(raw) > maxArtistImageBytes {
		return display, thumbnail, fmt.Errorf("artist image must be between 1 byte and 12 MiB")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png" && format != "gif") {
		return display, thumbnail, fmt.Errorf("artist image must be JPEG, PNG or GIF")
	}
	if config.Width < 16 || config.Height < 16 || config.Width*config.Height > maxArtistImagePixels {
		return display, thumbnail, fmt.Errorf("artist image dimensions are outside the accepted range")
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return display, thumbnail, fmt.Errorf("artist image could not be decoded")
	}
	source := image.NewNRGBA(decoded.Bounds())
	draw.Draw(source, source.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	if display, err = encodeArtistImage(downscale(source, ArtistDisplayWidth)); err != nil {
		return
	}
	thumbnail, err = encodeArtistImage(downscale(source, ArtistThumbnailWidth))
	return
}

// downscale averages each source pixel's area into the target grid. It never
// enlarges an image.
func downscale(src *image.NRGBA, width int) *image.NRGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw <= width {
		return src
	}
	height := max(1, (sh*width+sw/2)/sw)
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		y0, y1 := y*sh/height, max((y+1)*sh/height, y*sh/height+1)
		for x := 0; x < width; x++ {
			x0, x1 := x*sw/width, max((x+1)*sw/width, x*sw/width+1)
			var r, g, b, a, n uint64
			for sy := y0; sy < y1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < x1; sx++ {
					p := row[sx*4 : sx*4+4]
					alpha := uint64(p[3])
					r += uint64(p[0]) * alpha
					g += uint64(p[1]) * alpha
					b += uint64(p[2]) * alpha
					a += alpha
					n++
				}
			}
			o := dst.PixOffset(x, y)
			if a > 0 {
				dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2] = uint8(r/a), uint8(g/a), uint8(b/a)
			}
			dst.Pix[o+3] = uint8(a / n)
		}
	}
	return dst
}

func encodeArtistImage(img *image.NRGBA) (ArtistMedia, error) {
	var out bytes.Buffer
	media := ArtistMedia{Width: img.Bounds().Dx(), Height: img.Bounds().Dy()}
	if img.Opaque() {
		rgb := image.NewRGBA(img.Bounds())
		draw.Draw(rgb, rgb.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
		draw.Draw(rgb, rgb.Bounds(), img, img.Bounds().Min, draw.Over)
		media.ContentType = "image/jpeg"
		if err := jpeg.Encode(&out, rgb, &jpeg.Options{Quality: 85}); err != nil {
			return media, err
		}
	} else {
		media.ContentType = "image/png"
		if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, img); err != nil {
			return media, err
		}
	}
	media.data = out.Bytes()
	sum := sha256.Sum256(media.data)
	media.SHA256 = hex.EncodeToString(sum[:])
	return media, nil
}

// ArtistMedia returns a stored rendition only while a current profile of an
// artist with visible recordings references it.
func (s *Store) ArtistMedia(ctx context.Context, id string) (string, []byte, error) {
	if !ValidMediaID(id) {
		return "", nil, pgx.ErrNoRows
	}
	var contentType string
	var data []byte
	err := s.Core.Pool.QueryRow(ctx, `SELECT m.content_type,m.data FROM source_artist_media m WHERE m.sha256=$2
 AND EXISTS(SELECT 1 FROM source_artist_profiles p WHERE p.collection_id=$1
  AND (p.profile->'photo'->'image'->>'sha256'=$2 OR p.profile->'photo'->'thumbnail'->>'sha256'=$2)
  AND EXISTS(SELECT 1 FROM source_catalog_items i WHERE i.collection_id=p.collection_id AND NOT i.excluded AND lower(i.effective->>'creator')=p.source_artist))`, s.Collection, id).Scan(&contentType, &data)
	return contentType, data, err
}
