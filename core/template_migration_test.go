package core

import (
	"os"
	"path/filepath"
	"testing"

	"singbox-launcher/internal/constants"
	"singbox-launcher/internal/locale"
)

// lastTemplateVersion reads the persisted LastTemplateLauncherVersion marker.
func lastTemplateVersion(t *testing.T, root string) string {
	t.Helper()
	return locale.LoadSettings(filepath.Join(root, "bin")).LastTemplateLauncherVersion
}

// withAppVersion temporarily overrides constants.AppVersion for a test scope
// so the function-under-test reads the version we want without us touching
// the global outside the closure.
func withAppVersion(t *testing.T, v string, fn func()) {
	t.Helper()
	prev := constants.AppVersion
	constants.AppVersion = v
	t.Cleanup(func() { constants.AppVersion = prev })
	fn()
}

// makeTempLauncherDir builds an exec-dir-shaped layout: <root>/bin/, with
// optional pre-existing wizard_template.json and bin/settings.json contents.
func makeTempLauncherDir(t *testing.T, withTemplate bool, settingsJSON string) string {
	t.Helper()
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if withTemplate {
		if err := os.WriteFile(filepath.Join(binDir, "wizard_template.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write template: %v", err)
		}
	}
	if settingsJSON != "" {
		if err := os.WriteFile(filepath.Join(binDir, "settings.json"), []byte(settingsJSON), 0o644); err != nil {
			t.Fatalf("write settings: %v", err)
		}
	}
	return root
}

func templateExists(t *testing.T, root string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, "bin", "wizard_template.json"))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("stat template: %v", err)
	return false
}

func TestInvalidateTemplateIfStale_LegacyEmptyMarker_RemovesTemplate(t *testing.T) {
	// settings.json without last_template_launcher_version (legacy install).
	root := makeTempLauncherDir(t, true, `{"lang":"en"}`)
	withAppVersion(t, "v0.8.8", func() {
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("invalidate: %v", err)
		}
		if templateExists(t, root) {
			t.Fatal("expected template to be removed (legacy empty marker)")
		}
	})
}

func TestInvalidateTemplateIfStale_OlderVersion_RemovesTemplate(t *testing.T) {
	root := makeTempLauncherDir(t, true, `{"lang":"en","last_template_launcher_version":"v0.8.7"}`)
	withAppVersion(t, "v0.8.8", func() {
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("invalidate: %v", err)
		}
		if templateExists(t, root) {
			t.Fatal("expected template to be removed (last < current)")
		}
		// Marker must advance so the next launch doesn't re-invalidate.
		if got := lastTemplateVersion(t, root); got != "v0.8.8" {
			t.Fatalf("expected marker stamped to v0.8.8 after removal, got %q", got)
		}
	})
}

// Stale check on a version that has no template file on disk (already removed,
// or never present): we must still stamp the marker so a manually-placed
// template on the next launch is not wiped.
func TestInvalidateTemplateIfStale_StaleButNoFile_StampsMarker(t *testing.T) {
	root := makeTempLauncherDir(t, false, `{"lang":"en","last_template_launcher_version":"v0.8.7"}`)
	withAppVersion(t, "v0.8.8", func() {
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("invalidate: %v", err)
		}
		if got := lastTemplateVersion(t, root); got != "v0.8.8" {
			t.Fatalf("expected marker stamped to v0.8.8 even with no file, got %q", got)
		}
	})
}

// The reported bug: after an upgrade wipes the template, the user drops one in
// by hand. The next launch on the SAME version must keep it — invalidation is
// at-most-once per version.
func TestInvalidateTemplateIfStale_ManualTemplateSurvivesSecondLaunch(t *testing.T) {
	root := makeTempLauncherDir(t, true, `{"lang":"en","last_template_launcher_version":"v0.8.7"}`)
	withAppVersion(t, "v0.8.8", func() {
		// First launch on v0.8.8: stale → removed + marker stamped.
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("first invalidate: %v", err)
		}
		if templateExists(t, root) {
			t.Fatal("expected template removed on first launch")
		}
		// User places a template by hand.
		if err := os.WriteFile(filepath.Join(root, "bin", "wizard_template.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("manual write: %v", err)
		}
		// Second launch on the same version: must NOT remove it again.
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("second invalidate: %v", err)
		}
		if !templateExists(t, root) {
			t.Fatal("manually-placed template must survive a second launch on the same version")
		}
	})
}

func TestInvalidateTemplateIfStale_SameVersion_KeepsTemplate(t *testing.T) {
	root := makeTempLauncherDir(t, true, `{"lang":"en","last_template_launcher_version":"v0.8.8"}`)
	withAppVersion(t, "v0.8.8", func() {
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("invalidate: %v", err)
		}
		if !templateExists(t, root) {
			t.Fatal("expected template to be kept (last == current)")
		}
	})
}

func TestInvalidateTemplateIfStale_NewerVersion_KeepsTemplate(t *testing.T) {
	// Downgrade scenario: last installed by a *newer* launcher than current.
	// Don't touch the file — user knows what they're doing.
	root := makeTempLauncherDir(t, true, `{"lang":"en","last_template_launcher_version":"v0.8.9"}`)
	withAppVersion(t, "v0.8.8", func() {
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("invalidate: %v", err)
		}
		if !templateExists(t, root) {
			t.Fatal("expected template to be kept on downgrade")
		}
	})
}

func TestInvalidateTemplateIfStale_DevBuild_SkipsEntirely(t *testing.T) {
	root := makeTempLauncherDir(t, true, `{"lang":"en","last_template_launcher_version":"v0.8.7"}`)
	for _, v := range []string{"v-local-test", "unnamed-dev", "v0.8.7-3-gabc1234-dirty"} {
		v := v
		t.Run(v, func(t *testing.T) {
			withAppVersion(t, v, func() {
				if err := InvalidateTemplateIfStale(root); err != nil {
					t.Fatalf("invalidate: %v", err)
				}
				if !templateExists(t, root) {
					t.Fatal("expected dev build to leave template alone")
				}
			})
		})
	}
}

func TestInvalidateTemplateIfStale_NoTemplate_NoOp(t *testing.T) {
	// Fresh install: settings.json may or may not exist, no template file.
	// Should not error.
	root := makeTempLauncherDir(t, false, "")
	withAppVersion(t, "v0.8.8", func() {
		if err := InvalidateTemplateIfStale(root); err != nil {
			t.Fatalf("invalidate: %v", err)
		}
	})
}
