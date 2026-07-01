// node_edit_dialog.go — per-node outbound JSON editor dialog (SPEC 063).
//
// Two tabs: Form (convenient toggles/selects for common sing-box fields) and
// JSON (full raw editor). Edits are stored as a shallow override map in
// Source.NodeOverrides[tag] and merged onto node.Outbound at GenerateNodeJSON time.
package tabs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"singbox-launcher/internal/locale"
)

// uTLS fingerprint options (sing-box accepted values).
var utlsFingerprints = []string{
	"", "random", "randomized",
	"chrome", "firefox", "safari", "ios",
	"android", "edge", "360", "qq",
}

// showNodeEditDialog opens a window to edit the outbound JSON for a single node.
func showNodeEditDialog(
	parent fyne.Window,
	nodeTag string,
	currentOutbound map[string]interface{},
	currentOverride map[string]interface{},
	onSave func(newOverride map[string]interface{}),
) {
	app := fyne.CurrentApp()
	if app == nil {
		return
	}

	win := app.NewWindow(locale.Tf("wizard.node_edit.title", nodeTag))

	// Effective body = factory + override merged.
	effective := mergeMaps(currentOutbound, currentOverride)
	nodeScheme := schemeFromOutbound(effective) // "hysteria2", "vless", "vmess", "trojan", etc.

	// ---- Form tab ----
	formContent := container.NewVBox()

	// Mux section (universal).
	muxData := mapFromOutbound(effective, "mux")
	muxEnabled := boolFromMap(muxData, "enabled")
	muxConcStr := fmt.Sprintf("%v", valueFromMap(muxData, "concurrency", -1))
	muxXUDPStr := fmt.Sprintf("%v", valueFromMap(muxData, "xudpConcurrency", 8))

	muxToggle := widget.NewCheck(locale.T("wizard.node_edit.mux_enabled"), nil)
	muxToggle.SetChecked(muxEnabled)

	muxConcEntry := widget.NewEntry()
	muxConcEntry.SetText(muxConcStr)
	muxConcEntry.SetPlaceHolder("-1")
	if !muxEnabled {
		muxConcEntry.Disable()
	}

	muxXUDPEntry := widget.NewEntry()
	muxXUDPEntry.SetText(muxXUDPStr)
	muxXUDPEntry.SetPlaceHolder("8")
	if !muxEnabled {
		muxXUDPEntry.Disable()
	}

	muxToggle.OnChanged = func(on bool) {
		if on {
			muxConcEntry.Enable()
			muxXUDPEntry.Enable()
		} else {
			muxConcEntry.Disable()
			muxXUDPEntry.Disable()
		}
	}

	muxContent := container.NewVBox(
		muxToggle,
		widget.NewLabel(locale.T("wizard.node_edit.mux_concurrency")),
		muxConcEntry,
		widget.NewLabel(locale.T("wizard.node_edit.mux_xudp")),
		muxXUDPEntry,
	)

	// ---- Bandwidth (hysteria2 / any) ----
	upStr := fmt.Sprintf("%v", valueFromMap(effective, "up_mbps", 0))
	downStr := fmt.Sprintf("%v", valueFromMap(effective, "down_mbps", 0))

	upEntry := widget.NewEntry()
	upEntry.SetText(upStr)
	upEntry.SetPlaceHolder("0")

	downEntry := widget.NewEntry()
	downEntry.SetText(downStr)
	downEntry.SetPlaceHolder("0")

	bwContent := container.NewVBox(
		widget.NewLabel(locale.T("wizard.node_edit.up_mbps")),
		upEntry,
		widget.NewLabel(locale.T("wizard.node_edit.down_mbps")),
		downEntry,
	)

	// ---- TLS section (fingerprint + insecure) ----
	tlsData := mapFromOutbound(effective, "tls")
	utlsData := mapFromMap(tlsData, "utls")
	currentFP := stringFromMap(utlsData, "fingerprint")
	if currentFP == "" {
		// Fallback: try "fp" at top level (some legacy schemes).
		currentFP = stringFromMap(effective, "fp")
	}
	insecureVal := boolFromMap(tlsData, "insecure")

	fpSelect := widget.NewSelect(utlsFingerprints, nil)
	fpSelect.SetSelected(currentFP)
	if fpSelect.Selected != currentFP {
		// If the current value isn't in the list, add it temporarily.
		fpSelect.Options = append([]string{currentFP}, utlsFingerprints...)
		fpSelect.SetSelected(currentFP)
	}

	insecureToggle := widget.NewCheck(locale.T("wizard.node_edit.tls_insecure"), nil)
	insecureToggle.SetChecked(insecureVal)

	tlsContent := container.NewVBox(
		widget.NewLabel(locale.T("wizard.node_edit.tls_fingerprint")),
		fpSelect,
		insecureToggle,
	)

	// ---- Protocol-specific ----
	protoContent := container.NewVBox()

	// Obfs for hysteria2.
	if nodeScheme == "hysteria2" {
		obfsData := mapFromOutbound(effective, "obfs")
		obfsType := stringFromMap(obfsData, "type")
		obfsEnabled := obfsType != ""
		obfsPassword := stringFromMap(obfsData, "password")

		obfsToggle := widget.NewCheck(locale.T("wizard.node_edit.obfs_salamander"), nil)
		obfsToggle.SetChecked(obfsEnabled)

		obfsPassEntry := widget.NewEntry()
		obfsPassEntry.SetText(obfsPassword)
		if !obfsEnabled {
			obfsPassEntry.Disable()
		}

		obfsToggle.OnChanged = func(on bool) {
			if on {
				obfsPassEntry.Enable()
			} else {
				obfsPassEntry.Disable()
			}
		}

		protoContent.Add(obfsToggle)
		protoContent.Add(widget.NewLabel(locale.T("wizard.node_edit.obfs_password")))
		protoContent.Add(obfsPassEntry)
	}

	// Transport for VLESS/VMess/Trojan.
	if nodeScheme == "vless" || nodeScheme == "vmess" || nodeScheme == "trojan" {
		currentTransport := stringFromMap(effective, "transport")
		transports := []string{"", "tcp", "ws", "grpc", "http", "httpupgrade"}
		transportSelect := widget.NewSelect(transports, nil)
		transportSelect.SetSelected(currentTransport)

		protoContent.Add(widget.NewLabel(locale.T("wizard.node_edit.transport")))
		protoContent.Add(transportSelect)

		// TCP Fast Open (tfo) for VLESS/VMess/Trojan.
		tfoVal := boolFromMap(effective, "tcp_fast_open")
		tfoToggle := widget.NewCheck(locale.T("wizard.node_edit.tcp_fast_open"), nil)
		tfoToggle.SetChecked(tfoVal)
		protoContent.Add(tfoToggle)
	}

	formContent.Add(widget.NewSeparator())
	formContent.Add(widget.NewLabel(locale.T("wizard.node_edit.section_mux")))
	formContent.Add(muxContent)
	formContent.Add(widget.NewSeparator())
	formContent.Add(widget.NewLabel(locale.T("wizard.node_edit.section_bandwidth")))
	formContent.Add(bwContent)
	formContent.Add(widget.NewSeparator())
	formContent.Add(widget.NewLabel(locale.T("wizard.node_edit.section_tls")))
	formContent.Add(tlsContent)
	if len(protoContent.Objects) > 0 {
		formContent.Add(widget.NewSeparator())
		formContent.Add(widget.NewLabel(locale.T("wizard.node_edit.section_protocol")))
		formContent.Add(protoContent)
	}

	formScroll := container.NewScroll(formContent)
	formScroll.SetMinSize(fyne.NewSize(480, 380))

	// ---- Raw JSON tab ----
	displayJSON, _ := json.MarshalIndent(effective, "", "  ")
	rawEntry := widget.NewMultiLineEntry()
	rawEntry.SetText(string(displayJSON))
	rawEntry.Wrapping = fyne.TextWrapOff
	rawEntry.SetMinRowsVisible(18)
	rawScroll := container.NewScroll(rawEntry)
	rawScroll.SetMinSize(fyne.NewSize(560, 400))

	// Error label.
	errLabel := widget.NewLabel("")
	errLabel.Wrapping = fyne.TextWrapWord
	errLabel.Importance = widget.DangerImportance
	errLabel.Hide()

	// ---- Sync helpers ----
	// syncFormToRaw: read form values, build a merged outbound, write to rawEntry.
	syncFormToRaw := func() {
		out := make(map[string]interface{})
		for k, v := range effective {
			out[k] = v
		}

		// Mux
		if muxToggle.Checked {
			muxObj := map[string]interface{}{
				"enabled": true,
			}
			if c, err := strconv.Atoi(strings.TrimSpace(muxConcEntry.Text)); err == nil {
				muxObj["concurrency"] = c
			}
			if x, err := strconv.Atoi(strings.TrimSpace(muxXUDPEntry.Text)); err == nil {
				muxObj["xudpConcurrency"] = x
			}
			out["mux"] = muxObj
		} else {
			delete(out, "mux")
		}

		// Bandwidth
		if up, err := strconv.Atoi(strings.TrimSpace(upEntry.Text)); err == nil {
			out["up_mbps"] = up
		}
		if down, err := strconv.Atoi(strings.TrimSpace(downEntry.Text)); err == nil {
			out["down_mbps"] = down
		}

		// TLS
		if tls, ok := out["tls"].(map[string]interface{}); ok {
			if fpSelect.Selected != "" {
				if utls, ok := tls["utls"].(map[string]interface{}); ok {
					utls["enabled"] = true
					utls["fingerprint"] = fpSelect.Selected
				} else {
					tls["utls"] = map[string]interface{}{
						"enabled":     true,
						"fingerprint": fpSelect.Selected,
					}
				}
			} else {
				if utls, ok := tls["utls"].(map[string]interface{}); ok {
					utls["enabled"] = false
					utls["fingerprint"] = ""
				}
			}
			tls["insecure"] = insecureToggle.Checked
		}

		// Protocol: obfs
		if nodeScheme == "hysteria2" {
			obfsToggleVal := false
			for _, obj := range protoContent.Objects {
				if c, ok := obj.(*widget.Check); ok && c.Text == locale.T("wizard.node_edit.obfs_salamander") {
					obfsToggleVal = c.Checked
					break
				}
			}
			if obfsToggleVal {
				obfsObj := map[string]interface{}{"type": "salamander"}
				passText := ""
				for _, obj := range protoContent.Objects {
					if e, ok := obj.(*widget.Entry); ok {
						passText = strings.TrimSpace(e.Text)
						break
					}
				}
				if passText != "" {
					obfsObj["password"] = passText
				}
				out["obfs"] = obfsObj
			} else {
				delete(out, "obfs")
			}
		}

		// Protocol: transport
		if nodeScheme == "vless" || nodeScheme == "vmess" || nodeScheme == "trojan" {
			for _, obj := range protoContent.Objects {
				if s, ok := obj.(*widget.Select); ok && strings.Contains(strings.Join(s.Options, ","), "tcp") {
					if s.Selected != "" {
						out["transport"] = s.Selected
					} else {
						delete(out, "transport")
					}
				}
				if c, ok := obj.(*widget.Check); ok && c.Text == locale.T("wizard.node_edit.tcp_fast_open") {
					out["tcp_fast_open"] = c.Checked
				}
			}
		}

		b, _ := json.MarshalIndent(out, "", "  ")
		rawEntry.SetText(string(b))
	}

	// syncRawToForm: parse rawEntry JSON and update form widgets.
	syncRawToForm := func(text string) {
		var edited map[string]interface{}
		if json.Unmarshal([]byte(text), &edited) != nil {
			return
		}
		// Mux
		if mux, ok := edited["mux"].(map[string]interface{}); ok {
			muxToggle.SetChecked(boolFromMap(mux, "enabled"))
			if c, ok := mux["concurrency"]; ok {
				muxConcEntry.SetText(fmt.Sprintf("%v", c))
			}
			if x, ok := mux["xudpConcurrency"]; ok {
				muxXUDPEntry.SetText(fmt.Sprintf("%v", x))
			}
		} else {
			muxToggle.SetChecked(false)
		}
		// Bandwidth
		upEntry.SetText(fmt.Sprintf("%v", valueFromMap(edited, "up_mbps", 0)))
		downEntry.SetText(fmt.Sprintf("%v", valueFromMap(edited, "down_mbps", 0)))
		// TLS
		if tls, ok := edited["tls"].(map[string]interface{}); ok {
			insecureToggle.SetChecked(boolFromMap(tls, "insecure"))
			if utls, ok := tls["utls"].(map[string]interface{}); ok {
				if fp, ok := utls["fingerprint"].(string); ok && fp != "" {
					fpSelect.SetSelected(fp)
				} else {
					fpSelect.SetSelected("")
				}
			}
		}
		// Protocol
		if nodeScheme == "vless" || nodeScheme == "vmess" || nodeScheme == "trojan" {
			transport := stringFromMap(edited, "transport")
			for _, obj := range protoContent.Objects {
				if s, ok := obj.(*widget.Select); ok && strings.Contains(strings.Join(s.Options, ","), "tcp") {
					s.SetSelected(transport)
				}
				if c, ok := obj.(*widget.Check); ok && c.Text == locale.T("wizard.node_edit.tcp_fast_open") {
					c.SetChecked(boolFromMap(edited, "tcp_fast_open"))
				}
			}
		}
	}

	// Bind JSON editor changes to form.
	rawEntry.OnChanged = func(text string) {
		syncRawToForm(text)
	}

	// After form changes, push to raw.
	pushForm := func() {
		syncFormToRaw()
	}
	// Wire form callbacks.
	muxToggle.OnChanged = func(v bool) {
		if v {
			muxConcEntry.Enable()
			muxXUDPEntry.Enable()
		} else {
			muxConcEntry.Disable()
			muxXUDPEntry.Disable()
		}
		_ = muxToggle
		_ = muxConcEntry
		_ = muxXUDPEntry
		pushForm()
	}
	muxConcEntry.OnChanged = func(s string) { pushForm() }
	muxXUDPEntry.OnChanged = func(s string) { pushForm() }
	upEntry.OnChanged = func(s string) { pushForm() }
	downEntry.OnChanged = func(s string) { pushForm() }
	fpSelect.OnChanged = func(s string) { pushForm() }
	insecureToggle.OnChanged = func(b bool) { pushForm() }

	// Protocol callbacks.
	for _, obj := range protoContent.Objects {
		if c, ok := obj.(*widget.Check); ok {
			orig := c.OnChanged
			c.OnChanged = func(v bool) {
				if orig != nil {
					orig(v)
				}
				pushForm()
			}
		}
		if s, ok := obj.(*widget.Select); ok {
			orig := s.OnChanged
			s.OnChanged = func(v string) {
				if orig != nil {
					orig(v)
				}
				pushForm()
			}
		}
	}

	// Compute override from rawEntry.
	computeOverride := func() (map[string]interface{}, error) {
		text := rawEntry.Text
		var edited map[string]interface{}
		if err := json.Unmarshal([]byte(text), &edited); err != nil {
			return nil, fmt.Errorf("%s: %w", locale.T("wizard.node_edit.error_parse"), err)
		}
		override := make(map[string]interface{})
		for k, v := range edited {
			factoryVal, exists := currentOutbound[k]
			if !exists || !jsonDeepEqual(factoryVal, v) {
				override[k] = v
			}
		}
		if len(override) == 0 {
			return nil, nil
		}
		return override, nil
	}

	// Save / Reset / Cancel.
	saveBtn := widget.NewButton(locale.T("wizard.node_edit.button_save"), func() {
		pushForm()
		override, err := computeOverride()
		if err != nil {
			errLabel.SetText(err.Error())
			errLabel.Show()
			return
		}
		onSave(override)
		win.Close()
	})
	saveBtn.Importance = widget.HighImportance

	resetBtn := widget.NewButton(locale.T("wizard.node_edit.button_reset"), func() {
		onSave(nil)
		win.Close()
	})

	cancelBtn := widget.NewButton(locale.T("wizard.outbound.button_cancel"), func() {
		win.Close()
	})

	buttonsRow := container.NewHBox(layout.NewSpacer(), cancelBtn, resetBtn, saveBtn)

	formTab := container.NewTabItem(locale.T("wizard.node_edit.tab_form"), formScroll)
	jsonTab := container.NewTabItem(locale.T("wizard.node_edit.tab_json"), rawScroll)
	tabs := container.NewAppTabs(formTab, jsonTab)

	hint := widget.NewLabel(locale.T("wizard.node_edit.hint"))
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance

	content := container.NewBorder(
		container.NewVBox(hint, errLabel),
		buttonsRow,
		nil,
		nil,
		tabs,
	)

	win.SetContent(content)
	win.Resize(fyne.NewSize(700, 620))
	win.CenterOnScreen()
	win.Show()
}

// ---- Helpers ----

func mergeMaps(base, over map[string]interface{}) map[string]interface{} {
	if over == nil {
		return base
	}
	out := make(map[string]interface{}, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func schemeFromOutbound(out map[string]interface{}) string {
	if obfstype, ok := out["obfs"].(map[string]interface{}); ok {
		if t, _ := obfstype["type"].(string); t == "salamander" {
			// Has salamander obfs — likely hysteria2.
		}
	}
	if up, ok := out["up_mbps"]; ok && up != nil {
		// Has up_mbps — likely hysteria2.
	}
	if _, ok := out["password"]; ok {
		return "hysteria2"
	}
	if _, ok := out["uuid"]; ok {
		if _, ok := out["alter_id"]; ok {
			return "vmess"
		}
		return "vless"
	}
	if _, ok := out["method"]; ok {
		return "ss"
	}
	if _, ok := out["transport"]; ok {
		return "trojan"
	}
	// Guess from mux presence (hysteria2 always has mux populated by parser).
	if _, ok := out["mux"]; ok {
		return "hysteria2"
	}
	return ""
}

func mapFromOutbound(out map[string]interface{}, key string) map[string]interface{} {
	if v, ok := out[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func mapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func boolFromMap(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func stringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func valueFromMap(m map[string]interface{}, key string, fallback interface{}) interface{} {
	if m == nil {
		return fallback
	}
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}

func jsonDeepEqual(a, b interface{}) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ja) == string(jb)
}
