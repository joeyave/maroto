package gopdf

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // registers the JPEG decoder used to read image dimensions
	_ "image/png"  // registers the PNG decoder used to read image dimensions

	"github.com/johnfercher/maroto/v2/internal/providers/gopdf/gopdfwrapper"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// pixelsToMillimeters converts image pixels into millimeters at 72 DPI, the resolution
// used to size images (one pixel is one point).
const pixelsToMillimeters = 25.4 / 72.0

type Image struct {
	pdf  gopdfwrapper.Gopdf
	math core.Math
}

// NewImage create an Image.
func NewImage(pdf gopdfwrapper.Gopdf, math core.Math) *Image {
	return &Image{
		pdf,
		math,
	}
}

// GetImageInfo returns the dimensions of the image, in millimeters at 72 DPI.
func (s *Image) GetImageInfo(img *entity.Image, _ extension.Type) (*entity.Dimensions, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(img.Bytes))
	if err != nil {
		return nil, fmt.Errorf("could not decode image: %w", err)
	}

	return &entity.Dimensions{
		Width:  float64(config.Width) * pixelsToMillimeters,
		Height: float64(config.Height) * pixelsToMillimeters,
	}, nil
}

// Add use a byte array to add image to PDF.
func (s *Image) Add(img *entity.Image, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, extension extension.Type, flow bool,
) error {
	dimensions, err := s.GetImageInfo(img, extension)
	if err != nil {
		return err
	}

	return s.addImageToPdf(img, dimensions, cell, margins, prop, flow)
}

func (s *Image) addImageToPdf(img *entity.Image, info *entity.Dimensions, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, flow bool,
) error {
	// footprint is the area the image occupies on the page, so a quarter turn swaps its sides.
	footprint := info
	if prop.IsQuarterTurn() {
		footprint = &entity.Dimensions{Width: info.Height, Height: info.Width}
	}

	dimensions := s.math.Resize(footprint, cell.GetDimensions(), prop.Percent, prop.JustReferenceWidth)

	rectCell := &entity.Cell{X: prop.Left, Y: prop.Top, Width: dimensions.Width, Height: dimensions.Height}

	if prop.Center {
		rectCell = s.math.GetInnerCenterCell(dimensions, cell.GetDimensions())
	}

	x := cell.X + rectCell.X + margins.Left
	y := cell.Y + rectCell.Y + margins.Top
	w := rectCell.Width
	h := rectCell.Height

	if prop.RotationAngle == 0 {
		return s.pdf.Image(img.Bytes, x, y, w, h, flow)
	}

	imageWidth, imageHeight := w, h
	if prop.IsQuarterTurn() {
		imageWidth, imageHeight = h, w
	}

	return s.addRotatedImageToPdf(img, float64(prop.RotationAngle), x+w/2, y+h/2, imageWidth, imageHeight)
}

// addRotatedImageToPdf draws an unrotated image of the given size centered on (centerX, centerY)
// and rotates it around that point using PDF matrix transformations.
func (s *Image) addRotatedImageToPdf(img *entity.Image, angle, centerX, centerY, width, height float64) error {
	s.pdf.Rotate(angle, centerX, centerY)
	err := s.pdf.Image(img.Bytes, centerX-width/2, centerY-height/2, width, height, false)
	s.pdf.RotateReset()

	return err
}
