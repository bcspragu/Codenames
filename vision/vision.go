package vision

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"sort"

	codenames "github.com/bcspragu/Codenames"
	"github.com/bcspragu/Codenames/dict"
	"golang.org/x/net/context"
	"google.golang.org/api/option"

	vision "cloud.google.com/go/vision/apiv1"
	visionpb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

// Converter handles turning a board picture into a text-board.
type Converter struct {
	client *vision.ImageAnnotatorClient
	dict   *dict.Dictionary
}

// New builds a new Converter
func New(ctx context.Context, opts ...option.ClientOption) (*Converter, error) {
	client, err := vision.NewImageAnnotatorClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate Cloud Vision client: %v", err)
	}

	return &Converter{
		client: client,
	}, nil
}

// BoardFromReader reads an image and extracts a board from it.
func (c *Converter) BoardFromReader(ctx context.Context, r io.Reader) (*codenames.Board, error) {
	img, err := vision.NewImageFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read img from reader: %v", err)
	}

	resp, err := c.client.AnnotateImage(ctx, &visionpb.AnnotateImageRequest{
		Image: img,
		Features: []*visionpb.Feature{
			&visionpb.Feature{Type: visionpb.Feature_TEXT_DETECTION},
		},
		ImageContext: &visionpb.ImageContext{
			LanguageHints: []string{"en"}, // English words only
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to annotate image: %v", err)
	}
	var annos []*visionpb.EntityAnnotation
	// Filter out the "words" we can't find in the dictionary. At the very least,
	// this will handle the "return all the words as one entry" response entry.
	for _, an := range resp.TextAnnotations {
		if !c.dict.Valid(an.Description) {
			continue
		}
		annos = append(annos, an)
	}
	annos = filterByCenters(annos)
	return annosToBoard(annos), nil
}

// filterByHeight tries to group words by the height of the bounding box.
func filterByHeight(annos []*visionpb.EntityAnnotation) []*visionpb.EntityAnnotation {
	var validAnnos []*visionpb.EntityAnnotation
	for tol := 1; tol < 100; tol++ {
		hm := make(map[int][]*visionpb.EntityAnnotation)
		for _, an := range annos {
			ht := round(height(an.BoundingPoly), tol)
			hm[ht] = append(hm[ht], an)
		}
		for _, ans := range hm {
			if len(ans) >= codenames.Size {
				validAnnos = ans
				break
			}
		}
	}
	return validAnnos
}

// filterByCenters tries to group words by their centers, aligning them to an
// increasingly large grid. When it finds a 5x5 grid, it prints it out.
func filterByCenters(annos []*visionpb.EntityAnnotation) []*visionpb.EntityAnnotation {
	var validAnnos []*visionpb.EntityAnnotation
	for tol := 1; tol < 100; tol++ {
		xm := make(map[int][]*visionpb.EntityAnnotation)
		ym := make(map[int][]*visionpb.EntityAnnotation)
		for _, an := range annos {
			x, y := xyFromPoly(an.BoundingPoly)
			xr, yr := round(x, tol), round(y, tol)
			xm[xr] = append(xm[xr], an)
			ym[yr] = append(ym[yr], an)
		}
		var vx, vy []int
		for x, ans := range xm {
			if len(ans) == codenames.Rows {
				vx = append(vx, x)
			}
		}
		for y, ans := range ym {
			if len(ans) == codenames.Columns {
				vy = append(vy, y)
			}
		}
		if len(vx) == codenames.Columns && len(vy) == codenames.Rows {
			sort.Ints(vx)
			sort.Ints(vy)
			for _, y := range vy {
				for _, x := range vx {
					for _, an1 := range ym[y] {
						for _, an2 := range xm[x] {
							if an1.Description == an2.Description {
								validAnnos = append(validAnnos, an1)
							}
						}
					}
				}
			}
			break
		}
	}
	return validAnnos
}

func annosToBoard(annos []*visionpb.EntityAnnotation) *codenames.Board {
	cds := make([]codenames.Card, len(annos))
	for i, an := range annos {
		cds[i].Codename = an.Description
	}
	return &codenames.Board{Cards: cds}
}

func round(x, nearest int) int {
	return int((float64(x)+float64(nearest)/2.0)/float64(nearest)) * nearest
}

func height(poly *visionpb.BoundingPoly) int {
	if len(poly.Vertices) == 0 {
		return 0
	}
	max, min := poly.Vertices[0].Y, poly.Vertices[0].Y
	for _, v := range poly.Vertices {
		if v.Y > max {
			max = v.Y
		}
		if v.Y < min {
			min = v.Y
		}
	}
	return int(max - min)
}

func xyFromPoly(poly *visionpb.BoundingPoly) (int, int) {
	var x, y, n int
	for _, v := range poly.Vertices {
		x += int(v.X)
		y += int(v.Y)
		n++
	}
	return x / n, y / n
}

// processImage attempts to do some preprocessing to only identify words in the
// image that are actual codenames. This approach didn't work well on sample
// images though, so we do processing based on geometry and text content of the
// vision response.
func processImage(r io.Reader) (io.Reader, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, alpha := img.At(x, y).RGBA()
			rd, gn, bl, al := uint16(red), uint16(green), uint16(blue), uint16(alpha)
			c := color.RGBA64{rd, gn, bl, al}
			if !blackAndWhite(rd, gn, bl, 25000) {
				c.R = 65535
				c.G = 65535
				c.B = 65535
			}
			out.Set(x, y, c)
		}
	}
	pr, pw := io.Pipe()
	go func() {
		jpeg.Encode(pw, out, &jpeg.Options{Quality: 100})
		pw.Close()
	}()
	return pr, nil
}

func blackAndWhite(r, g, b uint16, x int) bool {
	d := uint16(x)
	if r > d && 65535-r < d {
		return false
	}

	if g > d && 65535-g < d {
		return false
	}

	if b > d && 65535-b < d {
		return false
	}

	return true
}

func absDiff(x, y uint16) int {
	d := int64(x) - int64(y)
	if d > 0 {
		return int(d)
	}
	return int(-d)
}
