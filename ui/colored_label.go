package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ColoredLabel is a custom widget that displays text with a colored background
type ColoredLabel struct {
	widget.BaseWidget
	text      string
	bgColor   color.Color
	textColor color.Color
	textObj   *canvas.Text
	bgRect    *canvas.Rectangle
	container *fyne.Container
}

// NewColoredLabel creates a new colored label
func NewColoredLabel(text string, bgColor, textColor color.Color) *ColoredLabel {
	cl := &ColoredLabel{
		text:      text,
		bgColor:   bgColor,
		textColor: textColor,
	}
	cl.ExtendBaseWidget(cl)
	return cl
}

// CreateRenderer implements fyne.Widget
func (cl *ColoredLabel) CreateRenderer() fyne.WidgetRenderer {
	cl.textObj = canvas.NewText(cl.text, cl.textColor)
	cl.textObj.TextStyle = fyne.TextStyle{}
	cl.textObj.Alignment = fyne.TextAlignLeading

	cl.bgRect = canvas.NewRectangle(cl.bgColor)

	cl.container = container.NewStack(cl.bgRect, cl.textObj)

	return &coloredLabelRenderer{
		label:     cl,
		container: cl.container,
		bgRect:    cl.bgRect,
		textObj:   cl.textObj,
	}
}

// SetText updates the label text
func (cl *ColoredLabel) SetText(text string) {
	cl.text = text
	if cl.textObj != nil {
		cl.textObj.Text = text
		cl.textObj.Refresh()
	}
}

// SetColors updates the label colors
func (cl *ColoredLabel) SetColors(bgColor, textColor color.Color) {
	cl.bgColor = bgColor
	cl.textColor = textColor
	if cl.bgRect != nil {
		cl.bgRect.FillColor = bgColor
		cl.bgRect.Refresh()
	}
	if cl.textObj != nil {
		cl.textObj.Color = textColor
		cl.textObj.Refresh()
	}
}

// coloredLabelRenderer implements fyne.WidgetRenderer
type coloredLabelRenderer struct {
	label     *ColoredLabel
	container *fyne.Container
	bgRect    *canvas.Rectangle
	textObj   *canvas.Text
}

func (r *coloredLabelRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *coloredLabelRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

func (r *coloredLabelRenderer) Refresh() {
	r.textObj.Text = r.label.text
	r.textObj.Color = r.label.textColor
	r.bgRect.FillColor = r.label.bgColor
	r.textObj.Refresh()
	r.bgRect.Refresh()
}

func (r *coloredLabelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.container}
}

func (r *coloredLabelRenderer) Destroy() {
	// Nothing to destroy
}
