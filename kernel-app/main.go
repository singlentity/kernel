package main

import (
	"syscall"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"

	webview2 "github.com/jchv/go-webview2"
)

//go:embed ui/app.html ui/app.js fonts/Inter-Regular.otf fonts/Inter-Medium.otf fonts/Inter-SemiBold.otf fonts/Inter-Bold.otf fonts/Inter-ExtraBold.otf
var assets embed.FS

func startServer() string {
	mux := http.NewServeMux()
	read := func(p string) []byte { b, _ := assets.ReadFile(p); return b }
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		switch {
		case path == "" || path == "index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(read("ui/app.html"))
		case path == "app.js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Write(read("ui/app.js"))
		case strings.HasPrefix(path, "fonts/"):
			if b, err := fs.ReadFile(assets, path); err == nil {
				w.Header().Set("Content-Type", "font/otf")
				w.Write(b)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go http.Serve(ln, mux)
	return fmt.Sprintf("http://%s/", ln.Addr().String())
}

func main() {
	initLog()

	// Тихий режим для задачи «Авто-очистка» (без окна)
	for _, a := range osArgs() {
		if a == "--autoclean" {
			logLine("INFO", "Авто-очистка по расписанию")
			cleanTemp()
			logLine("OK", "Авто-очистка завершена")
			return
		}
	}

	// Прогреваем «тяжёлые» данные в фоне, пока создаётся окно — вкладки откроются мгновенно
	go func() { defer func() { recover() }(); apiSysInfo(); refreshInstalled() }()

	url := startServer()
	winW, winH := windowSize()

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug: false,
		WindowOptions: webview2.WindowOptions{
			Title:  "KERNEL — System Optimizer",
			Width:  winW,
			Height: winH,
			Center: true,
			IconId: 1, // ресурс иконки из winres (.syso)
		},
	})
	if w == nil {
		showFatal("Не удалось запустить окно. Установите «Microsoft Edge WebView2 Runtime».")
		return
	}
	defer w.Destroy()

	w.SetSize(900, 600, webview2.HintMin)

	// ---- привязка Go-функций к интерфейсу ----
	bind := func(name string, f interface{}) {
		if err := w.Bind(name, f); err != nil {
			logLine("ERR", "bind "+name+": "+err.Error())
		}
	}
	bind("sysInfo", apiSysInfo)
	bind("balance", apiBalance)
	bind("checkUpdate", apiCheckUpdate)
	bind("appVersion", apiAppVersion)
	bind("vbsStatus", apiVbsStatus)
	bind("vbsRevert", apiVbsRevert)
	bind("tweaks", apiTweaks)
	bind("applyTweaks", apiApplyTweaks)
	bind("applyAll", apiApplyAll)
	bind("revertAll", apiRevertAll)
	bind("cleanScan", apiCleanScan)
	bind("clean", apiClean)
	bind("uwpList", apiUwpList)
	bind("removeUwp", apiRemoveUwp)
	bind("removeComp", apiRemoveComp)
	bind("appsList", apiAppsList)
	bind("install", apiInstall)
	bind("downloadAll", apiDownloadAll)
	bind("defenderStatus", apiDefenderStatus)
	bind("openDefender", apiOpenDefender)
	bind("driverList", apiDriverList)
	bind("updateDrivers", apiUpdateDrivers)
	bind("backupDrivers", apiBackupDrivers)
	bind("logLines", apiLogLines)
	bind("toggleAutoClean", apiToggleAutoClean)
	bind("edgeStatus", apiEdgeStatus)
	bind("openUrl", apiOpenUrl)
	bind("launchApp", apiLaunchApp)
	bind("reboot", apiReboot)
	bind("freshList", apiFreshList)
	bind("freshApply", apiFreshApply)
	bind("setWallpaper", apiSetWallpaper)
	bind("pinInfo", apiPinInfo)

	w.Navigate(url)
	w.Run()
}

func osArgs() []string { return osArgsImpl }


// Размер окна от экрана пользователя: ~85% ширины и высоты, но не меньше минимума и не больше экрана
func windowSize() (uint, uint) {
	user32 := syscall.NewLazyDLL("user32.dll")
	gsm := user32.NewProc("GetSystemMetrics")
	sw, _, _ := gsm.Call(0) // SM_CXSCREEN
	sh, _, _ := gsm.Call(1) // SM_CYSCREEN
	w, h := uint(float64(sw)*0.85), uint(float64(sh)*0.85)
	if w > 1700 {
		w = 1700
	}
	if h > 1000 {
		h = 1000
	}
	if w < 1000 {
		w = uint(sw)
	}
	if h < 640 {
		h = uint(sh)
	}
	return w, h
}
