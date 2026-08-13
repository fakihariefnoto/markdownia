package main

import (
	"context"
	"runtime"

	"github.com/anofac/markdownia/internal/binding"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var ctx = context.Background()

// buildMenu constructs the exact native menu from desktop/tasks/02-window-
// shell.md. Every item dispatches the same action as its shortcut via the
// frontend's action registry; menu items disable rather than disappear.
func buildMenu(app *application.App, native *binding.NativeService) {
	menu := app.NewMenu()

	// Markdownia (macOS only).
	if runtime.GOOS == "darwin" {
		appMenu := menu.AddSubmenu("Markdownia")
		appMenu.Add("About Markdownia").OnClick(func(_ *application.Context) {
			dispatch("open-about")
		})
		appMenu.Add("Check for Updates…").OnClick(func(_ *application.Context) {
			_, _ = native.CheckForUpdates(ctx, "anofac/markdownia")
		})
		appMenu.Add("Settings").SetAccelerator("CmdOrCtrl+,").OnClick(func(_ *application.Context) {
			dispatch("settings")
		})
		appMenu.AddSeparator()
		appMenu.Add("Hide Markdownia").OnClick(func(_ *application.Context) { app.Hide() })
		appMenu.Add("Quit Markdownia").SetAccelerator("CmdOrCtrl+Q").OnClick(func(_ *application.Context) {
			app.Quit()
		})
	}

	// File.
	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Import Folder…").SetAccelerator("CmdOrCtrl+O").OnClick(func(_ *application.Context) {
		dispatch("import-folder")
	})
	fileMenu.Add("Import Git Repository…").SetAccelerator("CmdOrCtrl+Shift+O").OnClick(func(_ *application.Context) {
		dispatch("import-git")
	})
	fileMenu.Add("Import Zip…").OnClick(func(_ *application.Context) {
		dispatch("import-zip")
	})
	fileMenu.AddSeparator()
	fileMenu.Add("New Collection…").SetAccelerator("CmdOrCtrl+N").OnClick(func(_ *application.Context) {
		dispatch("new-collection")
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Export as PDF…").SetAccelerator("CmdOrCtrl+P").OnClick(func(_ *application.Context) {
		dispatch("export-pdf")
	})
	fileMenu.Add("Export as HTML…").OnClick(func(_ *application.Context) {
		dispatch("export-html")
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Close Tab").SetAccelerator("CmdOrCtrl+W").OnClick(func(_ *application.Context) {
		dispatch("close-tab")
	})
	fileMenu.Add("Close All Tabs").OnClick(func(_ *application.Context) {
		dispatch("close-all-tabs")
	})
	if runtime.GOOS != "darwin" {
		fileMenu.AddSeparator()
		fileMenu.Add("Settings").SetAccelerator("Ctrl+,").OnClick(func(_ *application.Context) {
			dispatch("settings")
		})
		fileMenu.Add("Exit").SetAccelerator("Ctrl+Q").OnClick(func(_ *application.Context) {
			app.Quit()
		})
	}

	// Edit — uses Wails' role menu so Cut/Copy/Paste/Select All reach the
	// focused webview control (a custom menu without roles breaks Cmd+V).
	menu.Append(application.NewMenuFromItems(application.NewEditMenu()))

	// Find + Search Library are app-specific; add them to the Edit menu.
	editMenu := menu.AddSubmenu("Find")
	editMenu.Add("Find in Document").SetAccelerator("CmdOrCtrl+F").OnClick(func(_ *application.Context) {
		dispatch("find-in-document")
	})
	editMenu.Add("Find Next").SetAccelerator("CmdOrCtrl+G").OnClick(func(_ *application.Context) {
		dispatch("find-next")
	})
	editMenu.Add("Find Previous").SetAccelerator("CmdOrCtrl+Shift+G").OnClick(func(_ *application.Context) {
		dispatch("find-previous")
	})
	editMenu.AddSeparator()
	editMenu.Add("Search Library").SetAccelerator("CmdOrCtrl+Shift+F").OnClick(func(_ *application.Context) {
		dispatch("search-library")
	})

	// View.
	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Toggle Sidebar").SetAccelerator("CmdOrCtrl+B").OnClick(func(_ *application.Context) {
		dispatch("toggle-sidebar")
	})
	viewMenu.Add("Toggle Outline").SetAccelerator("CmdOrCtrl+Shift+B").OnClick(func(_ *application.Context) {
		dispatch("toggle-outline")
	})
	viewMenu.Add("Split View").SetAccelerator("CmdOrCtrl+\\").OnClick(func(_ *application.Context) {
		dispatch("toggle-split")
	})
	viewMenu.AddSeparator()

	readingTheme := viewMenu.AddSubmenu("Reading Theme")
	for _, theme := range []string{"Paper", "Sepia", "Solarized Light", "Nord", "Dracula", "Gruvbox"} {
		name := theme
		readingTheme.Add(name).OnClick(func(_ *application.Context) {
			dispatchValue("set-reading-theme", slug(name))
		})
	}

	appearance := viewMenu.AddSubmenu("Appearance")
	for _, mode := range []string{"Light", "Dark", "System"} {
		m := mode
		appearance.Add(m).OnClick(func(_ *application.Context) {
			dispatchValue("set-appearance", slug(m))
		})
	}

	accent := viewMenu.AddSubmenu("Accent")
	for _, a := range []string{"Teal", "Indigo", "Forest", "Copper", "Plum", "Slate"} {
		name := a
		accent.Add(name).OnClick(func(_ *application.Context) {
			dispatchValue("set-accent", slug(name))
		})
	}

	viewMenu.AddSeparator()
	font := viewMenu.AddSubmenu("Reading Font")
	font.Add("Sans").OnClick(func(_ *application.Context) { dispatchValue("set-reading-font", "sans") })
	font.Add("Serif").OnClick(func(_ *application.Context) { dispatchValue("set-reading-font", "serif") })
	viewMenu.Add("Zoom In").SetAccelerator("CmdOrCtrl++").OnClick(func(_ *application.Context) {
		dispatch("zoom-in")
	})
	viewMenu.Add("Zoom Out").SetAccelerator("CmdOrCtrl+-").OnClick(func(_ *application.Context) {
		dispatch("zoom-out")
	})
	viewMenu.Add("Actual Size").SetAccelerator("CmdOrCtrl+0").OnClick(func(_ *application.Context) {
		dispatch("zoom-reset")
	})

	// Go.
	goMenu := menu.AddSubmenu("Go")
	goMenu.Add("Back").SetAccelerator("CmdOrCtrl+[").OnClick(func(_ *application.Context) {
		dispatch("nav-back")
	})
	goMenu.Add("Forward").SetAccelerator("CmdOrCtrl+]").OnClick(func(_ *application.Context) {
		dispatch("nav-forward")
	})
	goMenu.AddSeparator()
	goMenu.Add("Command Palette").SetAccelerator("CmdOrCtrl+K").OnClick(func(_ *application.Context) {
		dispatch("command-palette")
	})
	goMenu.Add("Library Home").SetAccelerator("CmdOrCtrl+Shift+H").OnClick(func(_ *application.Context) {
		dispatch("library-home")
	})
	goMenu.Add("Annotations").OnClick(func(_ *application.Context) {
		dispatch("annotations")
	})

	// Source.
	sourceMenu := menu.AddSubmenu("Source")
	sourceMenu.Add("Refresh Current Source").SetAccelerator("CmdOrCtrl+R").OnClick(func(_ *application.Context) {
		dispatch("refresh-source")
	})
	sourceMenu.Add("Reveal in Finder").OnClick(func(_ *application.Context) {
		dispatch("reveal-source")
	})
	sourceMenu.AddSeparator()
	sourceMenu.Add("Rename Source…").OnClick(func(_ *application.Context) {
		dispatch("rename-source")
	})
	sourceMenu.Add("Relocate Source…").OnClick(func(_ *application.Context) {
		dispatch("relocate-source")
	})
	sourceMenu.Add("Delete Source…").OnClick(func(_ *application.Context) {
		dispatch("delete-source")
	})

	// Window (macOS).
	if runtime.GOOS == "darwin" {
		winMenu := menu.AddSubmenu("Window")
		winMenu.Add("Minimize").SetAccelerator("CmdOrCtrl+M").OnClick(func(_ *application.Context) {
			dispatch("window-minimize")
		})
		winMenu.Add("Zoom").OnClick(func(_ *application.Context) {
			dispatch("window-zoom")
		})
		winMenu.Add("Bring All to Front").OnClick(func(_ *application.Context) {})
	}

	// Help.
	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("Markdownia Help").OnClick(func(_ *application.Context) {
		dispatch("open-help")
	})
	helpMenu.Add("Keyboard Shortcuts").OnClick(func(_ *application.Context) {
		dispatch("help-shortcuts")
	})
	helpMenu.AddSeparator()
	helpMenu.Add("Reveal Log File").OnClick(func(_ *application.Context) {
		dispatch("reveal-log")
	})
	if runtime.GOOS != "darwin" {
		helpMenu.AddSeparator()
		helpMenu.Add("Check for Updates…").OnClick(func(_ *application.Context) {
			_, _ = native.CheckForUpdates(ctx, "anofac/markdownia")
		})
		helpMenu.Add("About").OnClick(func(_ *application.Context) {
			dispatch("open-about")
		})
	}

	app.Menu.Set(menu)
}

// slug converts a menu label to the frontend action value.
func slug(s string) string {
	switch s {
	case "Solarized Light":
		return "solarized"
	default:
		return lower(s)
	}
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// dispatch sends an action name to the frontend, mirroring the shortcut
// bindings in web/src/core/shortcuts.js.
func dispatch(action string) {
	execJS(`window.dispatchEvent(new CustomEvent('menu-action', { detail: ${JSON.stringify('` + action + `')} }))`)
}

// dispatchValue sends an action plus a value.
func dispatchValue(action, value string) {
	execJS(`window.dispatchEvent(new CustomEvent('menu-action', { detail: { action: ${JSON.stringify('` + action + `')}, value: ${JSON.stringify('` + value + `')} } }))`)
}

// execJS runs JS in the webview, dispatching a menu action the frontend's
// action registry handles.
func execJS(js string) {
	if app := application.Get(); app != nil {
		if win, ok := app.Window.Get("main"); ok {
			if w, ok2 := win.(*application.WebviewWindow); ok2 {
				w.ExecJS(js)
			}
		}
	}
}
