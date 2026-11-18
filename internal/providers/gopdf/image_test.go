package gopdf_test

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/johnfercher/maroto/v2/internal/fixture"
	"github.com/johnfercher/maroto/v2/internal/math"
	"github.com/johnfercher/maroto/v2/internal/providers/gopdf"
	"github.com/johnfercher/maroto/v2/mocks"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
)

// pngEntity creates a valid PNG of 72x144 pixels, which is 25.4x50.8 millimeters at 72 DPI.
func pngEntity(t *testing.T) entity.Image {
	t.Helper()
	return pngEntityOfSize(t, 72, 144)
}

func pngEntityOfSize(t *testing.T, width, height int) entity.Image {
	t.Helper()

	var buf bytes.Buffer
	err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height)))
	assert.NoError(t, err)

	return entity.Image{Bytes: buf.Bytes(), Extension: extension.Png}
}

func TestNewImage(t *testing.T) {
	t.Parallel()
	image := gopdf.NewImage(mocks.NewGopdf(t), mocks.NewMath(t))

	assert.NotNil(t, image)
	assert.Equal(t, "*gopdf.Image", fmt.Sprintf("%T", image))
}

func TestImage_Add(t *testing.T) {
	t.Parallel()
	t.Run("when image cannot be decoded, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		img := fixture.ImageEntity()

		sut := gopdf.NewImage(mocks.NewGopdf(t), mocks.NewMath(t))

		// Act
		err := sut.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.NotNil(t, err)
	})
	t.Run("when pdf cannot draw the image, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		img := pngEntity(t)
		drawErr := errors.New("draw error")

		pdf := mocks.NewGopdf(t)
		pdf.EXPECT().Image(img.Bytes, 30.0, 35.0, mock.Anything, mock.Anything, false).Return(drawErr)

		sut := gopdf.NewImage(pdf, math.New())

		// Act
		err := sut.Add(&img, &cell, &margins, &rect, img.Extension, false)

		// Assert
		assert.ErrorIs(t, err, drawErr)
	})
	t.Run("when prop is not center, should work properly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		img := pngEntity(t)

		pdf := mocks.NewGopdf(t)
		pdf.EXPECT().Image(img.Bytes, 30.0, 35.0, mock.Anything, mock.Anything, true).Return(nil)

		sut := gopdf.NewImage(pdf, math.New())

		// Act
		err := sut.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.Nil(t, err)
	})
	t.Run("when prop is center, should work properly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		rect.Center = true
		img := pngEntity(t)

		pdf := mocks.NewGopdf(t)
		pdf.EXPECT().Image(img.Bytes, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).Return(nil)

		sut := gopdf.NewImage(pdf, math.New())

		// Act
		err := sut.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.Nil(t, err)
	})
}

func TestImage_Add_Rotation(t *testing.T) {
	t.Parallel()
	// Cell 100x150 at (10, 15), margins 10, rect Left/Top 10 and Percent 98, image 200x100.
	cases := []struct {
		name          string
		angle         int
		center        bool
		expectedAngle float64
		pivotX        float64
		pivotY        float64
		x, y, w, h    float64
	}{
		{
			name: "when rotation angle is 180, should rotate around the image center without swapping sides",
			// footprint 98x49 at (30, 35)
			angle: 180, expectedAngle: 180, pivotX: 79, pivotY: 59.5, x: 30, y: 35, w: 98, h: 49,
		},
		{
			name: "when rotation angle is 90 and not centered, should place the rotated footprint at the rect origin",
			// footprint 73.5x147 at (30, 35)
			angle: 90, expectedAngle: 90, pivotX: 66.75, pivotY: 108.5, x: -6.75, y: 71.75, w: 147, h: 73.5,
		},
		{
			name: "when rotation angle is 270 and centered, should center the rotated footprint in the cell",
			// footprint 73.5x147 at (33.25, 26.5)
			angle: 270, center: true, expectedAngle: 270, pivotX: 70, pivotY: 100, x: -3.5, y: 63.25, w: 147, h: 73.5,
		},
		{
			name: "when rotation angle is -90, should be handled as 270",
			// footprint 73.5x147 at (30, 35)
			angle: -90, expectedAngle: 270, pivotX: 66.75, pivotY: 108.5, x: -6.75, y: 71.75, w: 147, h: 73.5,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Arrange
			cell := fixture.CellEntity()
			margins := fixture.MarginsEntity()
			rect := fixture.RectProp()
			rect.RotationAngle = c.angle
			rect.Center = c.center
			rect.MakeValid()
			img := pngEntityOfSize(t, 200, 100)

			var pivotX, pivotY, x, y, w, h float64

			pdf := mocks.NewGopdf(t)
			pdf.EXPECT().Rotate(c.expectedAngle, mock.Anything, mock.Anything).
				Run(func(_, px, py float64) { pivotX, pivotY = px, py })
			pdf.EXPECT().Image(img.Bytes, mock.Anything, mock.Anything, mock.Anything, mock.Anything, false).
				Run(func(_ []byte, ix, iy, iw, ih float64, _ bool) { x, y, w, h = ix, iy, iw, ih }).
				Return(nil)
			pdf.EXPECT().RotateReset()

			sut := gopdf.NewImage(pdf, math.New())

			// Act
			err := sut.Add(&img, &cell, &margins, &rect, img.Extension, true)

			// Assert
			assert.Nil(t, err)
			const delta = 1e-9
			assert.InDelta(t, c.pivotX, pivotX, delta)
			assert.InDelta(t, c.pivotY, pivotY, delta)
			assert.InDelta(t, c.x, x, delta)
			assert.InDelta(t, c.y, y, delta)
			assert.InDelta(t, c.w, w, delta)
			assert.InDelta(t, c.h, h, delta)
		})
	}
}

func TestImage_GetImageInfo(t *testing.T) {
	t.Parallel()
	t.Run("when image cannot be decoded, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := fixture.ImageEntity()
		sut := gopdf.NewImage(mocks.NewGopdf(t), mocks.NewMath(t))

		// Act
		info, err := sut.GetImageInfo(&img, img.Extension)

		// Assert
		assert.Nil(t, info)
		assert.NotNil(t, err)
	})
	t.Run("when image is valid, should return its dimensions at 72 DPI", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := pngEntity(t)
		sut := gopdf.NewImage(mocks.NewGopdf(t), mocks.NewMath(t))

		// Act
		info, err := sut.GetImageInfo(&img, img.Extension)

		// Assert
		assert.Nil(t, err)
		assert.InDelta(t, 25.4, info.Width, 1e-9)
		assert.InDelta(t, 50.8, info.Height, 1e-9)
	})
}
