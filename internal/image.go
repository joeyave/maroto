package internal

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"

	"github.com/google/uuid"
	"github.com/johnfercher/maroto/internal/fpdf"
	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/props"
	"github.com/jung-kurt/gofpdf"
)

// Image is the abstraction which deals of how to add images in a PDF.
type Image interface {
	AddFromFile(path string, cell Cell, prop props.Rect) (err error)
	AddFromBase64(stringBase64 string, cell Cell, prop props.Rect, extension consts.Extension) (err error)
	AddFromReader(r io.Reader, cell Cell, prop props.Rect, extension consts.Extension) (err error)
}

type image struct {
	pdf  fpdf.Fpdf
	math Math
}

// NewImage create an Image.
func NewImage(pdf fpdf.Fpdf, math Math) *image {
	return &image{
		pdf,
		math,
	}
}

// AddFromFile open an image from disk and add to PDF.
func (s *image) AddFromFile(path string, cell Cell, prop props.Rect) error {
	info := s.pdf.RegisterImageOptions(path, gofpdf.ImageOptions{
		ReadDpi:   false,
		ImageType: "",
	})

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	s.addImageToPdf(path, info, cell, prop)
	return nil
}

// AddFromBase64 use a base64 string to add to PDF.
func (s *image) AddFromBase64(stringBase64 string, cell Cell, prop props.Rect, extension consts.Extension) error {
	imageID, _ := uuid.NewRandom()

	ss, _ := base64.StdEncoding.DecodeString(stringBase64)

	info := s.pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytes.NewReader(ss),
	)

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	s.addImageToPdf(imageID.String(), info, cell, prop)
	return nil
}

// AddFromReader use a reader to add to PDF.
func (s *image) AddFromReader(r io.Reader, cell Cell, prop props.Rect, extension consts.Extension) error {
	imageID, _ := uuid.NewRandom()

	info := s.pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		r,
	)

	if info == nil {
		return errors.New("could not register image options, maybe path/name is wrong")
	}

	s.addImageToPdf(imageID.String(), info, cell, prop)
	return nil
}

func (s *image) addImageToPdf(imageLabel string, info *gofpdf.ImageInfoType, cell Cell, prop props.Rect) {
	var x, y, w, h float64
	if prop.Center {
		x, y, w, h = s.math.GetRectCenterColProperties(info.Width(), info.Height(), cell.Width, cell.Height, cell.X, prop.Percent)
	} else {
		x, y, w, h = s.math.GetRectNonCenterColProperties(info.Width(), info.Height(), cell.Width, cell.Height, cell.X, prop)
	}
	if prop.RotationAngle != 0 {
		xDelta := 0.
		yDelta := 0.
		finalW := w
		finalH := h
		if prop.RotationAngle == 90 || prop.RotationAngle == 270 {
			var _, _, rotatedW, rotatedH float64
			if prop.Center {
				_, _, rotatedW, rotatedH = s.math.GetRectCenterColProperties(info.Height(), info.Width(), cell.Width, cell.Height, cell.X, prop.Percent)
			} else {
				_, _, rotatedW, rotatedH = s.math.GetRectNonCenterColProperties(info.Height(), info.Width(), cell.Width, cell.Height, cell.X, prop)
			}
			xDelta = (w - rotatedH) / 2
			yDelta = (h - rotatedW) / 2
			finalW = rotatedH
			finalH = rotatedW
		}
		centerX := x + w/2
		centerY := y + h/2 + cell.Y
		s.pdf.TransformBegin()
		s.pdf.TransformRotate(float64(prop.RotationAngle), centerX, centerY)
		s.pdf.ImageOptions(imageLabel, x+xDelta, y+cell.Y+yDelta, finalW, finalH, false, gofpdf.ImageOptions{
			AllowNegativePosition: true,
		}, 0, "")
		s.pdf.TransformEnd()
	} else {
		s.pdf.Image(imageLabel, x, y+cell.Y, w, h, false, "", 0, "")
	}
}
