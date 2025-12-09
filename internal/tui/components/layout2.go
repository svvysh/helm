package components

import (
	"image"

	"github.com/polarzero/helm/internal/tui/theme"
)

// Constraint represents a size rule for layout purposes.
type Constraint interface {
	Apply(size int) int
}

// Percent constrains a dimension to a percentage of the available span.
type Percent int

// Apply applies the percentage constraint.
func (p Percent) Apply(size int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return size
	}
	return size * int(p) / 100
}

// Ratio is a helper to express Percent as a fraction.
func Ratio(numerator, denominator int) Percent {
	if denominator == 0 {
		return 0
	}
	return Percent(numerator * 100 / denominator)
}

// Fixed constrains a span to an exact value (clamped to the available space).
type Fixed int

// Apply applies the fixed constraint.
func (f Fixed) Apply(size int) int {
	if f < 0 {
		return 0
	}
	if int(f) > size {
		return size
	}
	return int(f)
}

// SplitVertical divides an area into top and bottom rectangles using the
// provided constraint.
func SplitVertical(area image.Rectangle, constraint Constraint) (top image.Rectangle, bottom image.Rectangle) {
	height := min(constraint.Apply(area.Dy()), area.Dy())
	top = image.Rectangle{Min: area.Min, Max: image.Point{X: area.Max.X, Y: area.Min.Y + height}}
	bottom = image.Rectangle{Min: image.Point{X: area.Min.X, Y: area.Min.Y + height}, Max: area.Max}
	return
}

// SplitHorizontal divides an area into left and right rectangles using the
// provided constraint.
func SplitHorizontal(area image.Rectangle, constraint Constraint) (left image.Rectangle, right image.Rectangle) {
	width := min(constraint.Apply(area.Dx()), area.Dx())
	left = image.Rectangle{Min: area.Min, Max: image.Point{X: area.Min.X + width, Y: area.Max.Y}}
	right = image.Rectangle{Min: image.Point{X: area.Min.X + width, Y: area.Min.Y}, Max: area.Max}
	return
}

// CenterRect returns a rectangle of the given size centered inside area.
func CenterRect(area image.Rectangle, width, height int) image.Rectangle {
	centerX := area.Min.X + area.Dx()/2
	centerY := area.Min.Y + area.Dy()/2
	minX := centerX - width/2
	minY := centerY - height/2
	return image.Rect(minX, minY, minX+width, minY+height).Intersect(area)
}

// TopLeftRect positions a rectangle of size width x height in the top-left.
func TopLeftRect(area image.Rectangle, width, height int) image.Rectangle {
	return image.Rect(area.Min.X, area.Min.Y, area.Min.X+width, area.Min.Y+height).Intersect(area)
}

// TopCenterRect positions a rectangle of size width x height in the top-center.
func TopCenterRect(area image.Rectangle, width, height int) image.Rectangle {
	centerX := area.Min.X + area.Dx()/2
	minX := centerX - width/2
	return image.Rect(minX, area.Min.Y, minX+width, area.Min.Y+height).Intersect(area)
}

// TopRightRect positions a rectangle of size width x height in the top-right.
func TopRightRect(area image.Rectangle, width, height int) image.Rectangle {
	return image.Rect(area.Max.X-width, area.Min.Y, area.Max.X, area.Min.Y+height).Intersect(area)
}

// RightCenterRect positions a rectangle of size width x height in the right-center.
func RightCenterRect(area image.Rectangle, width, height int) image.Rectangle {
	centerY := area.Min.Y + area.Dy()/2
	minY := centerY - height/2
	return image.Rect(area.Max.X-width, minY, area.Max.X, minY+height).Intersect(area)
}

// LeftCenterRect positions a rectangle of size width x height in the left-center.
func LeftCenterRect(area image.Rectangle, width, height int) image.Rectangle {
	centerY := area.Min.Y + area.Dy()/2
	minY := centerY - height/2
	return image.Rect(area.Min.X, minY, area.Min.X+width, minY+height).Intersect(area)
}

// BottomLeftRect positions a rectangle of size width x height in the bottom-left.
func BottomLeftRect(area image.Rectangle, width, height int) image.Rectangle {
	return image.Rect(area.Min.X, area.Max.Y-height, area.Min.X+width, area.Max.Y).Intersect(area)
}

// BottomCenterRect positions a rectangle of size width x height in the bottom-center.
func BottomCenterRect(area image.Rectangle, width, height int) image.Rectangle {
	centerX := area.Min.X + area.Dx()/2
	minX := centerX - width/2
	return image.Rect(minX, area.Max.Y-height, minX+width, area.Max.Y).Intersect(area)
}

// BottomRightRect positions a rectangle of size width x height in the bottom-right.
func BottomRightRect(area image.Rectangle, width, height int) image.Rectangle {
	return image.Rect(area.Max.X-width, area.Max.Y-height, area.Max.X, area.Max.Y).Intersect(area)
}

// ViewArea returns the full window rectangle anchored at the origin with sane defaults.
func ViewArea(width, height int) image.Rectangle {
	h := height
	if h < 0 {
		h = 0
	}
	return image.Rect(0, 0, ViewWidth(width), h)
}

// ContentArea returns the rectangle representing the drawable content once
// global padding is removed. Width is clamped to ContentWidth.
func ContentArea(width, height int) image.Rectangle {
	view := ViewArea(width, height)
	contentWidth := ContentWidth(view.Dx())
	contentHeight := view.Dy() - theme.ViewTopPadding - theme.ViewBottomPadding
	if contentHeight < 0 {
		contentHeight = 0
	}
	return image.Rect(
		view.Min.X+theme.ViewHorizontalPadding,
		view.Min.Y+theme.ViewTopPadding,
		view.Min.X+theme.ViewHorizontalPadding+contentWidth,
		view.Min.Y+theme.ViewTopPadding+contentHeight,
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
