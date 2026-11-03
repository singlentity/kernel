package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Result — реальный итог действия для интерфейса
type Result struct {
	Ok  bool   `json:"ok"`
	Msg string `json:"msg"`
}

func okR(msg string) Result  { logLine("OK", msg); return Result{true, msg} }
func errR(msg string) Result { logLine("ERR", msg); return Result{false, msg} }

/* ================= Источники установщиков ================= */
type source struct {
	URL      string // прямая ссылка
	Repo     string // или GitHub-репозиторий (берём последний релиз)
	Asset    string // регулярка имени файла в релизе
	File     string // имя файла при сохранении
	Args     []string
	MSI      bool
	Interact bool // запуск обычным окном (без тихого режима)
	Portable bool   // zip без установщика: распаковать и запустить
	Fallback string // проверенная прямая ссылка, если GitHub не отдал версию
}

var sources = map[string]source{
	"npp":        {Repo: "notepad-plus-plus/notepad-plus-plus", Asset: `(?i)^npp\.[\d.]+\.Installer\.x64\.exe$`, File: "npp-setup.exe", Args: []string{"/S"}},
	"mpc":        {Repo: "clsid2/mpc-hc", Asset: `(?i)^MPC-HC\.[\d.]+\.x64\.exe$`, File: "mpc-setup.exe", Args: []string{"/VERYSILENT", "/NORESTART", "/SP-"}},
	"imageglass": {URL: "https://github.com/d2phap/ImageGlass/releases/download/8.12.4.30/ImageGlass_8.12.4.30_x64.msi", File: "imageglass-8.12.msi", MSI: true},
	"raven":      {Repo: "mjishnu/Raven", Asset: `(?i)win-x64-self-contained\.zip$`, File: "Raven.zip", Portable: true,
		Fallback: "https://github.com/mjishnu/Raven/releases/download/v1.1.0.0/Raven-v1.1.0.0-win-x64-self-contained.zip"},
	"firefox":    {URL: "https://download.mozilla.org/?product=firefox-latest-ssl&os=win64&lang=ru", File: "firefox-setup.exe", Args: []string{"/S"}},
	"chrome":     {URL: "https://dl.google.com/chrome/install/googlechromestandaloneenterprise64.msi", File: "chrome-setup.msi", MSI: true},
	"brave":      {Repo: "brave/brave-browser", Asset: `^BraveBrowserStandaloneSetup\.exe$`, File: "brave-setup.exe", Args: []string{"/silent", "/install"}},
	"yandex":     {URL: "https://browser.yandex.ru/download?os=win&full=1", File: "yandex-setup.exe", Args: []string{"--silent", "--do-not-launch-browser"}},
}

var httpc = &http.Client{Timeout: 15 * time.Minute}

// Ищем самую свежую версию файла в релизах GitHub (2 способа, без зависимости от одного API)
func githubAsset(repo, pattern string) (string, error) {
	re := regexp.MustCompile(pattern)
	type relT struct {
		Draft  bool `json:"draft"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	// 0) API: стабильный «latest»
	if req0, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases/latest", nil); req0 != nil {
		req0.Header.Set("User-Agent", "KERNEL")
		req0.Header.Set("Accept", "application/vnd.github+json")
		if resp0, err := httpc.Do(req0); err == nil {
			var r relT
			if resp0.StatusCode == 200 && json.NewDecoder(resp0.Body).Decode(&r) == nil {
				for _, as := range r.Assets {
					if re.MatchString(as.Name) {
						resp0.Body.Close()
						return as.URL, nil
					}
				}
			}
			resp0.Body.Close()
		}
	}
	// 1) API: список релизов, берём самый свежий с подходящим файлом (в т.ч. pre-release)
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases?per_page=15", nil)
	req.Header.Set("User-Agent", "KERNEL")
	req.Header.Set("Accept", "application/vnd.github+json")
	apiErr := ""
	if resp, err := httpc.Do(req); err != nil {
		apiErr = err.Error()
	} else {
		var rels []relT
		if resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&rels) == nil {
			for _, r := range rels {
				if r.Draft {
					continue
				}
				for _, as := range r.Assets {
					if re.MatchString(as.Name) {
						resp.Body.Close()
						return as.URL, nil
					}
				}
			}
		}
		apiErr = fmt.Sprintf("API ответил %d", resp.StatusCode)
		resp.Body.Close()
	}
	logLine("WARN", repo+": "+apiErr+", ищу через страницу релиза")

	// 2) Без API: /releases/latest -> тег -> список файлов релиза
	noRedirect := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req2, _ := http.NewRequest("GET", "https://github.com/"+repo+"/releases/latest", nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 KERNEL")
	resp2, err := noRedirect.Do(req2)
	if err != nil {
		return "", fmt.Errorf("GitHub недоступен: %v", err)
	}
	loc := resp2.Header.Get("Location")
	resp2.Body.Close()
	i := strings.LastIndex(loc, "/tag/")
	if i < 0 {
		return "", fmt.Errorf("не удалось узнать последнюю версию %s", repo)
	}
	tag := loc[i+5:]
	req3, _ := http.NewRequest("GET", "https://github.com/"+repo+"/releases/expanded_assets/"+tag, nil)
	req3.Header.Set("User-Agent", "Mozilla/5.0 KERNEL")
	resp3, err := httpc.Do(req3)
	if err != nil {
		return "", fmt.Errorf("GitHub недоступен: %v", err)
	}
	body, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	hrefs := regexp.MustCompile(`href="(/[^"]+/releases/download/[^"]+)"`).FindAllStringSubmatch(string(body), -1)
	for _, h := range hrefs {
		name := h[1][strings.LastIndex(h[1], "/")+1:]
		if re.MatchString(name) {
			return "https://github.com" + h[1], nil
		}
	}
	return "", fmt.Errorf("в релизе %s нет подходящего файла", tag)
}

func download(url, dst string) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 KERNEL")
	resp, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("сервер ответил %d", resp.StatusCode)
	}
	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil || n < 100*1024 {
		os.Remove(tmp)
		return fmt.Errorf("загрузка прервана")
	}
	return os.Rename(tmp, dst)
}

var desktopCache string

// Рабочий стол пользователя (учитывает перенос в OneDrive и русское имя папки)
func desktopDir() string {
	if desktopCache == "" {
		desktopCache = psLine(`[Environment]::GetFolderPath('Desktop')`)
		if desktopCache == "" {
			desktopCache = filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
		}
	}
	return desktopCache
}

// Куда KERNEL складывает скачанные программы: папка на рабочем столе
func downloadDir() string { return filepath.Join(desktopDir(), "KERNEL Программы") }

// localInstaller ищет уже скачанный/положенный установщик (папка на столе или Installers рядом с exe)
func localInstaller(id string) string {
	for _, dir := range []string{downloadDir(), installersDir()} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			n := strings.ToLower(e.Name())
			if e.IsDir() {
				continue
			}
			if strings.Contains(n, id) && (strings.HasSuffix(n, ".exe") || strings.HasSuffix(n, ".msi") || strings.HasSuffix(n, ".zip")) {
				return filepath.Join(dir, e.Name())
			}
		}
	}
	return ""
}

func fetchInstaller(id string) (string, error) {
	if p := localInstaller(id); p != "" {
		return p, nil
	}
	s, ok := sources[id]
	if !ok {
		return "", fmt.Errorf("нет источника")
	}
	url := s.URL
	if s.Repo != "" {
		u, err := githubAsset(s.Repo, s.Asset)
		if err != nil {
			if s.Fallback == "" {
				return "", err
			}
			logLine("WARN", "Последняя версия не определилась ("+err.Error()+"), беру проверенную ссылку")
			u = s.Fallback
		}
		url = u
	}
	os.MkdirAll(downloadDir(), 0755)
	dst := filepath.Join(downloadDir(), s.File)
	logLine("INFO", "Скачиваю "+s.File+" на рабочий стол…")
	if err := download(url, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// Распаковка портативной программы (Raven) и поиск её exe
func unzipTo(zipPath, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		p := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(dst)+string(os.PathSeparator)) {
			continue // защита от путей вида ../
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(p, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(p), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(p)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
func findExe(root, kw string) string {
	exact, partial := "", ""
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		n := strings.ToLower(info.Name())
		if n == kw+".exe" && exact == "" {
			exact = p
		} else if partial == "" && strings.HasSuffix(n, ".exe") && strings.Contains(n, kw) && !strings.Contains(n, "updater") {
			partial = p
		}
		return nil
	})
	if exact != "" {
		return exact
	}
	return partial
}
func ravenDir() string { return filepath.Join(downloadDir(), "Raven") }

func runInstaller(id, path string) error {
	s := sources[id]
	if strings.HasSuffix(strings.ToLower(path), ".msi") || s.MSI {
		_, err := run("msiexec.exe", "/i", path, "/qn", "/norestart")
		return err
	}
	if s.Interact {
		_, err := run("cmd", "/c", "start", "/wait", "", path)
		return err
	}
	_, err := run(path, s.Args...)
	return err
}

func apiInstall(id string) Result {
	a := findApp(id)
	if a == nil {
		return errR("Неизвестная программа")
	}
	name := a["name"].(string)
	if exe := appExe(id); exe != "" {
		desktopShortcut(name, exe)
		return okR(name + " уже установлен в " + filepath.Dir(exe) + ". Ярлык — на рабочем столе")
	}
	path, err := fetchInstaller(id)

	// Портативные программы (Raven): распаковать на рабочий стол и запустить
	if sources[id].Portable {
		if err != nil {
			return errR(name + ": не удалось скачать (" + err.Error() + ")")
		}
		logLine("INFO", "Распаковываю "+name+"…")
		if err := unzipTo(path, ravenDir()); err != nil {
			return errR(name + ": архив повреждён (" + err.Error() + ")")
		}
		os.Remove(path)
		exe := findExe(ravenDir(), "raven")
		if exe == "" {
			run("explorer.exe", ravenDir())
			return errR(name + ": распакован в «KERNEL Программы\\Raven», но exe не найден — открыл папку")
		}
		desktopShortcut(name, exe)
		run("explorer.exe", exe)
		return okR(name + " распакован в «KERNEL Программы\\Raven» на рабочем столе и запущен. Ярлык — на рабочем столе")
	}

	if err != nil {
		logLine("WARN", name+": не удалось скачать ("+err.Error()+"), пробую winget")
		wg, _ := a["winget"].(string)
		if wg == "" {
			return errR(name + ": не удалось скачать (" + err.Error() + ")")
		}
		ps(fmt.Sprintf(`winget install --id %s -e --silent --accept-source-agreements --accept-package-agreements --disable-interactivity`, wg))
	} else {
		logLine("INFO", "Устанавливаю "+name+"…")
		runInstaller(id, path)
	}
	for i := 0; i < 20; i++ {
		if exe := appExe(id); exe != "" {
			desktopShortcut(name, exe)
			return okR(name + " установлен в " + filepath.Dir(exe) + ". Ярлык — на рабочем столе")
		}
		time.Sleep(1500 * time.Millisecond)
	}
	return errR(name + ": установка не подтвердилась. Установщик лежит в «KERNEL Программы» на рабочем столе — можно запустить вручную")
}

func apiDownloadAll() Result {
	got, failed := 0, []string{}
	for id := range sources {
		if _, err := fetchInstaller(id); err != nil {
			failed = append(failed, id)
		} else {
			got++
		}
	}
	if len(failed) > 0 {
		return errR(fmt.Sprintf("Скачано %d, не удалось: %s", got, strings.Join(failed, ", ")))
	}
	return okR(fmt.Sprintf("Все установщики (%d) сохранены в папку «KERNEL Программы» на рабочем столе — теперь ставятся без интернета", got))
}

/* ================= Edge / Store / WebView2 ================= */
func edgeInstalled() bool {
	for _, p := range []string{os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application\msedge.exe`, os.Getenv("ProgramFiles") + `\Microsoft\Edge\Application\msedge.exe`} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
func webviewInstalled() bool {
	for _, p := range []string{os.Getenv("ProgramFiles(x86)") + `\Microsoft\EdgeWebView\Application`, os.Getenv("ProgramFiles") + `\Microsoft\EdgeWebView\Application`} {
		if entries, err := os.ReadDir(p); err == nil {
			for _, e := range entries {
				if e.IsDir() && len(e.Name()) > 0 && e.Name()[0] >= '0' && e.Name()[0] <= '9' {
					return true
				}
			}
		}
	}
	return false
}
func storeInstalled() bool {
	return psLine(`(Get-AppxPackage -AllUsers Microsoft.WindowsStore -ErrorAction SilentlyContinue | Measure-Object).Count`) != "0"
}

func apiEdgeStatus() map[string]bool {
	return map[string]bool{"edge": edgeInstalled(), "store": storeInstalled(), "webview": webviewInstalled()}
}

func uninstallChromium(folder, flag string) {
	roots := []string{os.Getenv("ProgramFiles(x86)") + `\Microsoft\` + folder + `\Application`, os.Getenv("ProgramFiles") + `\Microsoft\` + folder + `\Application`}
	for _, r := range roots {
		vers, _ := os.ReadDir(r)
		for _, v := range vers {
			setup := filepath.Join(r, v.Name(), "Installer", "setup.exe")
			if _, err := os.Stat(setup); err == nil {
				logLine("INFO", "Запуск штатного деинсталлятора: "+setup)
				run(setup, "--uninstall", flag, "--system-level", "--verbose-logging", "--force-uninstall")
			}
		}
	}
}

func apiRemoveComp(kind string) Result {
	switch kind {
	case "store":
		if !storeInstalled() {
			return okR("Microsoft Store уже удалён")
		}
		ps(`Get-AppxPackage -AllUsers *WindowsStore* | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue
Get-AppxPackage -AllUsers *StorePurchaseApp* | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue
Get-AppxProvisionedPackage -Online | Where-Object {$_.DisplayName -like '*WindowsStore*'} | Remove-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue`)
		if storeInstalled() {
			return errR("Microsoft Store: Windows не дала удалить пакет")
		}
		return okR("Microsoft Store удалён")
	case "edge":
		if !edgeInstalled() {
			return okR("Microsoft Edge уже удалён")
		}
		return removeEdgeFull()
	}
	return errR("неизвестный компонент")
}

/* ================= UWP ================= */
func apiRemoveUwp(ids []string) Result {
	logLine("INFO", fmt.Sprintf("=== Удаление приложений (%d) ===", len(ids)))
	removed, failed := 0, []string{}
	needExplorer := false
	for _, pkg := range ids {
		// Спец-пункты (не Appx) — через реестр
		if strings.HasPrefix(pkg, "__") {
			if removeSpecial(pkg) {
				removed++
				needExplorer = true
				logLine("OK", "Отключено: "+pkg)
			} else {
				failed = append(failed, pkg)
			}
			continue
		}
		p := strings.ReplaceAll(pkg, "'", "")
		ps(fmt.Sprintf(`$n='%s'
Get-AppxPackage -AllUsers -Name "*$n*" | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue
Get-AppxPackage -Name "*$n*" | Remove-AppxPackage -ErrorAction SilentlyContinue
Get-AppxProvisionedPackage -Online | Where-Object {$_.DisplayName -like "*$n*"} | Remove-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue`, p))
		left := psLine(fmt.Sprintf(`(Get-AppxPackage -AllUsers -Name "*%s*" | Measure-Object).Count`, p))
		if left == "0" {
			removed++
			logLine("OK", "Удалено: "+pkg)
		} else {
			failed = append(failed, pkg)
			logLine("ERR", "Не удалось удалить: "+pkg+" (защищён системой)")
		}
	}
	installedPkgs = nil
	if needExplorer {
		run("taskkill", "/f", "/im", "explorer.exe")
		exec.Command("explorer.exe").Start()
	}
	if len(failed) > 0 {
		return Result{removed > 0, fmt.Sprintf("Удалено %d, не удалось %d — подробности в Журнале", removed, len(failed))}
	}
	return Result{true, fmt.Sprintf("Удалено приложений: %d", removed)}
}

// Открыть ссылку в браузере по умолчанию (только https)
func apiOpenUrl(u string) bool {
	if !strings.HasPrefix(u, "https://") {
		return false
	}
	run("rundll32", "url.dll,FileProtocolHandler", u)
	return true
}


// Ярлык программы на рабочем столе
func desktopShortcut(name, exe string) {
	if exe == "" {
		return
	}
	lnk := filepath.Join(desktopDir(), name+".lnk")
	ps(fmt.Sprintf(`$s=(New-Object -ComObject WScript.Shell).CreateShortcut('%s');$s.TargetPath='%s';$s.WorkingDirectory='%s';$s.Save()`,
		strings.ReplaceAll(lnk, "'", "''"), strings.ReplaceAll(exe, "'", "''"), strings.ReplaceAll(filepath.Dir(exe), "'", "''")))
	logLine("OK", "Ярлык на рабочем столе: "+name)
}

// Запуск установленной программы (через Проводник — без прав администратора)
func apiLaunchApp(id string) Result {
	exe := appExe(id)
	if exe == "" {
		return errR("Программа не найдена на диске")
	}
	run("explorer.exe", exe)
	return Result{true, ""}
}

// Перезагрузка по кнопке из баннера (с задержкой, чтобы успеть сохраниться)
func apiReboot() bool {
	logLine("INFO", "Перезагрузка по запросу пользователя через 20 секунд")
	run("shutdown.exe", "/r", "/t", "20", "/c", "KERNEL: перезагрузка для применения изменений")
	return true
}

// Полное удаление Edge: собственный деинсталлятор Edge + Appx + остановка автообновления,
// чтобы Edge не вернулся. Защиту целостности системных файлов Windows не трогаем.
func removeEdgeFull() Result {
	logLine("INFO", "=== Полное удаление Microsoft Edge ===")

	// 1) Закрыть всё, что связано с Edge и его апдейтером
	for _, p := range []string{"msedge.exe", "msedgewebview2.exe", "MicrosoftEdgeUpdate.exe", "identity_helper.exe"} {
		run("taskkill", "/f", "/im", p)
	}

	// 2) Снять блокировку удаления в политике обновления Edge
	for _, root := range []string{`HKLM\SOFTWARE\Microsoft\EdgeUpdate`, `HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate`} {
		reg("add", root, "/v", "NoRemove", "/t", "REG_DWORD", "/d", "0", "/f")
		reg("add", root, "/v", "NoRemoveMsi", "/t", "REG_DWORD", "/d", "0", "/f")
	}

	// 3) Запустить собственный деинсталлятор Edge (все найденные версии, с system-level и без него)
	runEdgeUninstaller()

	// 3b) Если файлы остались — пробуем ЕС-режим: временно ставим европейский регион,
	// где Microsoft по закону (DMA) обязана разрешать удаление Edge её же деинсталлятором,
	// затем возвращаем регион. Защиту системных файлов не трогаем.
	if edgeInstalled() {
		oldNation := psLine(`(Get-ItemProperty 'HKCU:\Control Panel\International\Geo' -Name Nation -ErrorAction SilentlyContinue).Nation`)
		logLine("INFO", "Edge держится — пробую ЕС-режим удаления (регион временно = Германия)")
		reg("add", `HKCU\Control Panel\International\Geo`, "/v", "Nation", "/t", "REG_SZ", "/d", "94", "/f") // 94 = Германия (ЕС)
		for _, p := range []string{"msedge.exe", "MicrosoftEdgeUpdate.exe"} {
			run("taskkill", "/f", "/im", p)
		}
		runEdgeUninstaller()
		if oldNation != "" {
			reg("add", `HKCU\Control Panel\International\Geo`, "/v", "Nation", "/t", "REG_SZ", "/d", oldNation, "/f")
		}
	}

	// 4) Удалить Appx-версии Edge для всех пользователей и из образа
	ps(`Get-AppxPackage -AllUsers *MicrosoftEdge* | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue
Get-AppxPackage -AllUsers *Microsoft.Edge* | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue
Get-AppxProvisionedPackage -Online | Where-Object {$_.PackageName -like '*MicrosoftEdge*'} | ForEach-Object { Remove-AppxProvisionedPackage -Online -PackageName $_.PackageName -ErrorAction SilentlyContinue }`)

	// 5) Остановить и удалить службы и задачи автообновления Edge, чтобы он не вернулся
	for _, svc := range []string{"edgeupdate", "edgeupdatem", "MicrosoftEdgeElevationService"} {
		run("sc.exe", "stop", svc)
		run("sc.exe", "delete", svc)
	}
	ps(`Get-ScheduledTask -TaskName 'MicrosoftEdgeUpdate*' -ErrorAction SilentlyContinue | Unregister-ScheduledTask -Confirm:$false -ErrorAction SilentlyContinue`)

	// 6) Запретить Windows Update повторно ставить Edge
	for _, root := range []string{`HKLM\SOFTWARE\Microsoft\EdgeUpdate`, `HKLM\SOFTWARE\Policies\Microsoft\EdgeUpdate`} {
		reg("add", root, "/v", "DoNotUpdateToEdgeWithChromium", "/t", "REG_DWORD", "/d", "1", "/f")
		reg("add", root, "/v", "InstallDefault", "/t", "REG_DWORD", "/d", "0", "/f")
	}

	// 7) Убрать ярлыки Edge из меню Пуск и с рабочего стола
	ps(`Get-ChildItem -Path "$env:ProgramData\Microsoft\Windows\Start Menu","$env:APPDATA\Microsoft\Windows\Start Menu","$env:PUBLIC\Desktop","$env:USERPROFILE\Desktop" -Recurse -Filter '*Edge*.lnk' -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue`)

	time.Sleep(2 * time.Second)
	if edgeInstalled() {
		// Файлы удалить нельзя — блокируем сам запуск Edge штатным механизмом Windows (IFEO).
		// Затрагивает только msedge.exe; msedgewebview2.exe (на нём работает KERNEL) не трогаем.
		ifeo := `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\msedge.exe`
		reg("add", ifeo, "/v", "Debugger", "/t", "REG_SZ", "/d", "%windir%\\System32\\systray.exe", "/f")
		logLine("OK", "Запуск Edge заблокирован (IFEO). Отменить: удалить ключ msedge.exe в Image File Execution Options")
		logLine("WARN", "Edge: файлы ещё на месте — даже в ЕС-режиме сборка блокирует удаление")
		return Result{false, "Edge нейтрализован: он больше не запускается, автообновление и ярлыки убраны. Но файлы ваша сборка Windows удалять запрещает даже в ЕС-режиме. Стереть их силой можно только отключив защиту целостности Windows — это ломает обновления, и KERNEL такого не делает."}
	}
	logLine("OK", "Microsoft Edge полностью удалён")
	return okR("Microsoft Edge полностью удалён. Автообновление отключено, повторная установка через Windows Update заблокирована.")
}

func runEdgeUninstaller() {
	roots := []string{os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application`, os.Getenv("ProgramFiles") + `\Microsoft\Edge\Application`}
	for _, r := range roots {
		vers, _ := os.ReadDir(r)
		for _, v := range vers {
			setup := filepath.Join(r, v.Name(), "Installer", "setup.exe")
			if _, err := os.Stat(setup); err != nil {
				continue
			}
			logLine("INFO", "Деинсталлятор Edge: "+setup)
			// с правами системы и в пользовательском контексте — что из этого разрешит сборка
			run(setup, "--uninstall", "--msedge", "--system-level", "--verbose-logging", "--force-uninstall")
			run(setup, "--uninstall", "--msedge", "--force-uninstall")
		}
	}
}

// Удаление спец-пунктов (значок Spotlight, виджеты) через реестр
func removeSpecial(id string) bool {
	switch id {
	case "__spotlight":
		for _, root := range []string{
			`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel`,
			`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\ClassicStartMenu`,
		} {
			reg("add", root, "/v", "{2cc5ca98-6485-489a-920e-b3e88a6ccce3}", "/t", "REG_DWORD", "/d", "1", "/f")
		}
		return true
	case "__widgets":
		regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, "TaskbarDa", 0)
		regDword("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", 0)
		ps(`Get-AppxPackage -AllUsers *WebExperience* | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue`)
		return true
	}
	return false
}
