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

	// ping color thresholds
	pingGreen  = color.NRGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF} // <70ms — green
	pingYellow = color.NRGBA{R: 0xFF, G: 0x9F, B: 0x0A, A: 0xFF} // 70-105ms — yellow/orange
	pingRed    = color.NRGBA{R: 0xFF, G: 0x45, B: 0x3A, A: 0xFF} // >105ms — red

	pingLabel = "⬡" // symbol for ping line (matches ↓/↑ style)
)

// Widget is a ready-to-use Fyne widget that displays real-time download/upload
// speeds and server ping from the Clash API + TCP ping. Use Container() to embed it.
type Widget struct {
	cfg ClashConfig
	mon *Monitor

	dlText    *canvas.Text
	upText    *canvas.Text
	pingText  *canvas.Text
	proxyText *canvas.Text
	addrText  *canvas.Text
	infoText  *canvas.Text
	status    *canvas.Text
	refresh   *widget.Button

	// diagnostic expand/collapse
	detailBtn   *widget.Button
	detailBox   *fyne.Container
	detailsOpen bool

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

	// Ping line — ⬡ symbol + delay
	w.pingText = canvas.NewText(pingLabel+" N/A", inactiveColor)
	w.pingText.TextSize = 13

	// Diagnostic lines (hidden by default)
	w.proxyText = canvas.NewText("Proxy: —", inactiveColor)
	w.proxyText.TextSize = 10

	w.addrText = canvas.NewText("Ping: —", inactiveColor)
	w.addrText.TextSize = 10

	w.infoText = canvas.NewText("Info: starting...", inactiveColor)
	w.infoText.TextSize = 10

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
	// Ping row: ⬡ label + [▼] toggle
	w.detailBtn = widget.NewButton("▶", w.toggleDetails)
	w.detailBtn.Importance = widget.LowImportance

	pingRow := container.NewHBox(w.pingText, w.detailBtn)

	// Detail rows — start hidden
	w.proxyText.Hide()
	w.addrText.Hide()
	w.infoText.Hide()

	proxyRow := container.NewHBox(w.proxyText)
	addrRow := container.NewHBox(w.addrText)
	infoRow := container.NewHBox(w.infoText)

	w.detailBox = container.NewVBox(
		proxyRow,
		addrRow,
		infoRow,
	)
	w.detailBox.Hide()

	dlRow := container.NewHBox(w.dlText)
	upRow := container.NewHBox(w.upText)
	statusRow := container.NewHBox(w.status)

	ctrlRow := container.NewHBox(
		w.refresh,
	)

	body := container.NewVBox(
		container.NewPadded(pingRow),
		w.detailBox,
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

// toggleDetails expands/collapses the diagnostic detail panel.
func (w *Widget) toggleDetails() {
	w.detailsOpen = !w.detailsOpen
	if w.detailsOpen {
		w.detailBtn.SetText("▼")
		w.detailBox.Show()
	} else {
		w.detailBtn.SetText("▶")
		w.detailBox.Hide()
	}
}

// pingColor returns the color based on delay threshold.
func pingColor(delayMs int64) color.Color {
	if delayMs <= 0 {
		return inactiveColor
	}
	switch {
	case delayMs < 70:
		return pingGreen
	case delayMs < 105:
		return pingYellow
	default:
		return pingRed
	}
}

// statsLoop reads from the monitor channel and updates the UI labels.
func (w *Widget) statsLoop() {
	for stats := range w.mon.Stats() {
		// Clone to capture value
		s := stats
		fyne.Do(func() {
			w.dlText.Text = "↓ " + s.DownStr
			w.upText.Text = "↑ " + s.UpStr

			// Ping line with color
			pc := pingColor(s.ServerDelayMs)
			w.pingText.Text = pingLabel + " " + s.ServerDelayStr
			w.pingText.Color = pc

			// Update diagnostic info (hidden behind ▶ button)
			if s.ProxyTag != "" {
				w.proxyText.Text = "Proxy: " + s.ProxyTag
			} else {
				w.proxyText.Text = "Proxy: —"
			}
			if s.ProxyAddr != "" {
				statusLabel := "✓"
				if !s.PingOk {
					statusLabel = "✗"
				}
				w.addrText.Text = "Ping: " + s.ProxyAddr + " " + statusLabel
			} else {
				w.addrText.Text = "Ping: —"
			}
			if s.PingInfo != "" {
				w.infoText.Text = "Info: " + s.PingInfo
			} else {
				w.infoText.Text = "Info: —"
			}

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
			canvas.Refresh(w.pingText)
			canvas.Refresh(w.proxyText)
			canvas.Refresh(w.addrText)
			canvas.Refresh(w.infoText)
			canvas.Refresh(w.status)
			canvas.Refresh(w.detailBtn)
		})
	}
}
