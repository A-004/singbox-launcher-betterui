package traffic

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	activeColor   = color.NRGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF} // green
	inactiveColor = color.NRGBA{R: 0x8E, G: 0x8E, B: 0x93, A: 0xFF} // gray
)

// Widget is a ready-to-use Fyne widget that displays real-time download/upload
// speeds from the Clash API. Use Container() to embed it in your app.
type Widget struct {
	cfg    ClashConfig
	mon    *Monitor

	dlText  *canvas.Text
	upText  *canvas.Text
	status  *canvas.Text
	refresh *widget.Button

	compact bool
}

// NewWidget creates a traffic widget with the given config.
// The monitor starts automatically. Call Stop() to clean up.
func NewWidget(cfg ClashConfig) *Widget {
	w := &Widget{
		cfg: cfg,
	}
	w.mon = NewMonitor(cfg)

	// Download label
	w.dlText = canvas.NewText("↓ 0 B/s", activeColor)
	w.dlText.TextSize = 15
	w.dlText.TextStyle = fyne.TextStyle{Bold: false}

	// Upload label
	w.upText = canvas.NewText("↑ 0 B/s", inactiveColor)
	w.upText.TextSize = 15
	w.upText.TextStyle = fyne.TextStyle{Bold: false}

	// Status line
	w.status = canvas.NewText("Disconnected", inactiveColor)
	w.status.TextSize = 9

	// Refresh button
	w.refresh = widget.NewButton("↻", func() {
		w.refresh.Disable()
		w.mon.Stop()
		w.mon = NewMonitor(w.cfg)
		w.mon.Start()
		w.refresh.Enable()
	})
	w.refresh.Importance = widget.LowImportance

	w.mon.Start()
	go w.statsLoop()

	return w
}

// Stop stops the monitor and cleans up resources.
func (w *Widget) Stop() {
	if w.mon != nil {
		w.mon.Stop()
	}
}

// Container returns the root Fyne object for embedding.
func (w *Widget) Container() fyne.CanvasObject {
	dlRow := container.NewHBox(w.dlText)
	upRow := container.NewHBox(w.upText)
	statusRow := container.NewHBox(w.status)

	ctrlRow := container.NewHBox(
		w.refresh,
	)

	body := container.NewVBox(
		container.NewPadded(dlRow),
		container.NewPadded(upRow),
		statusRow,
		ctrlRow,
	)

	// Wrap in a subtle card
	bg := canvas.NewRectangle(color.NRGBA{R: 0x1C, G: 0x1C, B: 0x1E, A: 0xFF})
	bg.CornerRadius = 12
	border := canvas.NewRectangle(color.Transparent)
	border.CornerRadius = 12
	border.StrokeWidth = 0.5
	border.StrokeColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x14}

	return container.NewStack(bg, border, container.NewPadded(body))
}

// statsLoop reads from the monitor channel and updates the UI labels.
func (w *Widget) statsLoop() {
	for stats := range w.mon.Stats() {
		// Clone to capture value
		s := stats
		fyne.Do(func() {
			w.dlText.Text = "↓ " + s.DownStr
			w.upText.Text = "↑ " + s.UpStr

			if s.IsActive {
				w.dlText.Color = activeColor
				w.upText.Color = activeColor
				w.status.Text = "Connected"
				w.status.Color = activeColor
			} else {
				w.dlText.Color = inactiveColor
				w.upText.Color = inactiveColor
				w.status.Text = "Disconnected"
				w.status.Color = inactiveColor
			}

			canvas.Refresh(w.dlText)
			canvas.Refresh(w.upText)
			canvas.Refresh(w.status)
		})
	}
}
