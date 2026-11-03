package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

/* ===== Фоновые службы и задачи: отключение с точным откатом ===== */

const servicesPath = `SYSTEM\CurrentControlSet\Services`

// Исходный тип запуска служб, которые меняли твики (имя → Start), чтобы откат вернул именно его
var svcMu sync.Mutex

func svcBackupPath() string { return filepath.Join(kernDir, "services-backup.json") }

func svcLoadBackup() map[string]int {
	m := map[string]int{}
	if data, err := os.ReadFile(svcBackupPath()); err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}
func svcSaveBackup(m map[string]int) {
	if data, err := json.Marshal(m); err == nil {
		os.WriteFile(svcBackupPath(), data, 0644)
	}
}

// Раскрываем шаблоны: «GoogleUpdaterService*» → все службы с таким началом имени.
// Отсутствующие на этом ПК службы просто пропускаются.
func svcExpand(names []string) []string {
	var all []string
	res := []string{}
	for _, n := range names {
		if strings.HasSuffix(n, "*") {
			if all == nil {
				all = regSubKeys("HKLM", servicesPath)
			}
			pre := strings.ToLower(strings.TrimSuffix(n, "*"))
			for _, s := range all {
				if strings.HasPrefix(strings.ToLower(s), pre) {
					res = append(res, s)
				}
			}
		} else if regKeyExists("HKLM", servicesPath+`\`+n) {
			res = append(res, n)
		}
	}
	return res
}

// Экземпляры пользовательских служб (CDPUserSvc_1a2b3) — их тоже нужно остановить
func svcInstances(name string) []string {
	res := []string{}
	pre := strings.ToLower(name) + "_"
	for _, s := range regSubKeys("HKLM", servicesPath) {
		if strings.HasPrefix(strings.ToLower(s), pre) {
			res = append(res, s)
		}
	}
	return res
}

// mode: 3 — вручную (служба запустится, только когда её позовут), 4 — отключена
func svcDisable(names []string, mode int) {
	svcMu.Lock()
	defer svcMu.Unlock()
	b := svcLoadBackup()
	for _, s := range svcExpand(names) {
		cur, ok := regReadDword("HKLM", servicesPath+`\`+s, "Start")
		if !ok || int(cur) >= mode {
			continue // уже вручную/отключена — не трогаем
		}
		if _, saved := b[s]; !saved {
			b[s] = int(cur)
		}
		regDword("HKLM", servicesPath+`\`+s, "Start", mode)
		run("sc.exe", "stop", s)
		for _, inst := range svcInstances(s) {
			run("sc.exe", "stop", inst)
		}
		logLine("OK", "Служба "+s+" → "+map[int]string{3: "вручную", 4: "отключена"}[mode])
	}
	svcSaveBackup(b)
}

func svcRestore(names []string) {
	svcMu.Lock()
	defer svcMu.Unlock()
	b := svcLoadBackup()
	for _, s := range svcExpand(names) {
		orig, saved := b[s]
		if !saved {
			continue // не мы меняли — не трогаем
		}
		regDword("HKLM", servicesPath+`\`+s, "Start", orig)
		if orig == 2 {
			run("sc.exe", "start", s)
		}
		delete(b, s)
	}
	svcSaveBackup(b)
}

// Применено, если все найденные службы уже не запускаются автоматически
func svcAllAtLeast(names []string, mode int) bool {
	for _, s := range svcExpand(names) {
		if v, ok := regReadDword("HKLM", servicesPath+`\`+s, "Start"); ok && int(v) < mode {
			return false
		}
	}
	return true
}

/* ---- Задачи планировщика ---- */
func tasksSet(tasks []string, enable bool) {
	flag := "/disable"
	if enable {
		flag = "/enable"
	}
	for _, t := range tasks {
		if _, err := run("schtasks.exe", "/change", "/tn", t, flag); err == nil {
			logLine("OK", "Задача "+t+map[bool]string{true: " включена", false: " отключена"}[enable])
		}
	}
}

// Проверяем по первой существующей задаче (XML не зависит от языка Windows)
func tasksDisabled(tasks []string) bool {
	for _, t := range tasks {
		out, err := run("schtasks.exe", "/query", "/tn", t, "/xml")
		if err != nil {
			continue
		}
		return strings.Contains(out, "<Enabled>false</Enabled>")
	}
	return true
}

/* ---- Счётчик процессов (для наглядности на вкладке «Ускорение») ---- */
func apiProcCount() int {
	pids := make([]uint32, 4096)
	var n uint32
	if err := windows.EnumProcesses(pids, &n); err != nil {
		return 0
	}
	return int(n / 4)
}

/* ---- Наборы служб и задач для твиков ---- */
var (
	svcTelemetry  = []string{"InventorySvc", "dmwappushservice", "WerSvc", "PcaSvc", "whesvc", "diagnosticshub.standardcollector.service"}
	svcAI         = []string{"WSAIFabricSvc"}
	svcRemote     = []string{"TermService", "SessionEnv", "UmRdpService", "RemoteRegistry"}
	svcPhoneLink  = []string{"CDPSvc", "CDPUserSvc", "PhoneSvc", "MapsBroker", "lfsvc", "RetailDemo", "WalletService"}
	svcXbox       = []string{"XblAuthManager", "XblGameSave", "XboxNetApiSvc"} // XboxGipSvc не трогаем — нужен геймпадам
	svcPrint      = []string{"Spooler", "PrintNotify", "Fax"}
	svcSearch     = []string{"WSearch"}
	svcThirdParty = []string{"gupdate", "gupdatem", "GoogleUpdaterService*", "GoogleUpdaterInternalService*", "MozillaMaintenance",
		"AdobeARMservice", "YandexBrowserService", "GigabyteUpdateService", "BraveElevationService", "brave", "bravem", "edgeupdate", "edgeupdatem"}

	telemetryTasks = []string{
		`\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser`,
		`\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser Exp`,
		`\Microsoft\Windows\Application Experience\ProgramDataUpdater`,
		`\Microsoft\Windows\Application Experience\PcaPatchDbTask`,
		`\Microsoft\Windows\Customer Experience Improvement Program\Consolidator`,
		`\Microsoft\Windows\Customer Experience Improvement Program\UsbCeip`,
		`\Microsoft\Windows\Autochk\Proxy`,
		`\Microsoft\Windows\DiskDiagnostic\Microsoft-Windows-DiskDiagnosticDataCollector`,
		`\Microsoft\Windows\Feedback\Siuf\DmClient`,
		`\Microsoft\Windows\Feedback\Siuf\DmClientOnScenarioDownload`,
		`\Microsoft\Windows\Windows Error Reporting\QueueReporting`,
		`\Microsoft\Windows\Maps\MapsUpdateTask`,
	}
)

// Твики фоновой нагрузки (добавляются к основному списку в initTweaks)
func backgroundTweaks() []Tweak {
	const grp = "Фоновые процессы"
	const svcGrp = "Фоновые службы (по выбору)"
	return []Tweak{
		{ID: "teltasks", Group: grp, Name: "Отключить задачи телеметрии в планировщике",
			Desc: "Отключает фоновые задачи сбора данных — в первую очередь «Оценку совместимости» (CompatTelRunner.exe), которая периодически грузит процессор и диск, иногда прямо во время игры.", Recommended: true,
			apply:     func() { tasksSet(telemetryTasks, false) },
			revert:    func() { tasksSet(telemetryTasks, true) },
			isApplied: func() bool { return tasksDisabled(telemetryTasks) },
		},
		{ID: "telsvc", Group: grp, Name: "Отключить службы диагностики и отчётов об ошибках",
			Desc: "Инвентаризация совместимости, отчёты об ошибках, помощник совместимости программ и сбор диагностики. Обычный пользователь от них ничего не получает, а процессы висят постоянно.", Recommended: true,
			apply:     func() { svcDisable(svcTelemetry, 4) },
			revert:    func() { svcRestore(svcTelemetry) },
			isApplied: func() bool { return svcAllAtLeast(svcTelemetry, 4) },
		},
		{ID: "ai", Group: grp, Name: "Отключить Copilot, Recall и ИИ-компоненты Windows",
			Desc: "Запрещает Recall (снимки экрана для «поиска по истории») и Copilot, останавливает службу ИИ-компонентов Windows. Минус фоновые процессы и нагрузка на диск.", Recommended: true,
			apply: func() {
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Windows\WindowsAI`, "DisableAIDataAnalysis", 1)
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Windows\WindowsAI`, "AllowRecallEnablement", 0)
				regDword("HKCU", `SOFTWARE\Policies\Microsoft\Windows\WindowsCopilot`, "TurnOffWindowsCopilot", 1)
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, "ShowCopilotButton", 0)
				svcDisable(svcAI, 4)
			},
			revert: func() {
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Windows\WindowsAI`, "DisableAIDataAnalysis")
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Windows\WindowsAI`, "AllowRecallEnablement")
				regDel("HKCU", `SOFTWARE\Policies\Microsoft\Windows\WindowsCopilot`, "TurnOffWindowsCopilot")
				svcRestore(svcAI)
			},
			isApplied: func() bool {
				return dwordIs("HKLM", `SOFTWARE\Policies\Microsoft\Windows\WindowsAI`, "DisableAIDataAnalysis", 1)
			},
		},
		{ID: "widgets", Group: grp, Name: "Отключить виджеты (новости и погода)",
			Desc: "Полностью выключает панель виджетов: процессы Widgets.exe и WebExperienceHost перестают висеть в фоне и подкачивать новости. Значок на панели задач исчезнет.", Recommended: true,
			apply: func() {
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", 0)
				run("taskkill.exe", "/f", "/im", "Widgets.exe")
				run("taskkill.exe", "/f", "/im", "WidgetService.exe")
			},
			revert:    func() { regDel("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests") },
			isApplied: func() bool { return dwordIs("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", 0) },
		},
		{ID: "browserbg", Group: grp, Name: "Запретить браузерам работать после закрытия",
			Desc: "Edge и Chrome по умолчанию остаются в фоне после закрытия окна, а Edge ещё и предзапускается вместе с Windows («ускорение запуска»). Запрещаем — после закрытия браузер действительно закрывается.", Recommended: true,
			apply: func() {
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "StartupBoostEnabled", 0)
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "BackgroundModeEnabled", 0)
				regDword("HKLM", `SOFTWARE\Policies\Google\Chrome`, "BackgroundModeEnabled", 0)
			},
			revert: func() {
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "StartupBoostEnabled")
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "BackgroundModeEnabled")
				regDel("HKLM", `SOFTWARE\Policies\Google\Chrome`, "BackgroundModeEnabled")
			},
			isApplied: func() bool { return dwordIs("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "BackgroundModeEnabled", 0) },
		},
		{ID: "dop2p", Group: grp, Name: "Не раздавать обновления другим компьютерам",
			Desc: "Windows по умолчанию раздаёт скачанные обновления другим ПК через ваш интернет и диск (оптимизация доставки). Оставляем только обычное скачивание с серверов Microsoft — сеть не занята во время игры.", Recommended: true,
			apply:     func() { regDword("HKLM", `SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`, "DODownloadMode", 0) },
			revert:    func() { regDel("HKLM", `SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`, "DODownloadMode") },
			isApplied: func() bool { return dwordIs("HKLM", `SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`, "DODownloadMode", 0) },
		},

		// ---- По выбору: полезно не всем, поэтому по умолчанию выключено ----
		{ID: "svcupdaters", Group: svcGrp, Name: "Службы-обновлялки сторонних программ — только по требованию",
			Desc: "Google, Яндекс, Brave, Mozilla, Adobe, Gigabyte держат свои службы обновления запущенными постоянно. Переводим их в режим «вручную»: не висят в фоне, но программы по-прежнему обновляются при запуске.",
			apply:     func() { svcDisable(svcThirdParty, 3) },
			revert:    func() { svcRestore(svcThirdParty) },
			isApplied: func() bool { return svcAllAtLeast(svcThirdParty, 3) },
		},
		{ID: "svcremote", Group: svcGrp, Name: "Отключить удалённый рабочий стол",
			Desc: "Службы удалённого доступа к этому ПК (RDP) и удалённого реестра. Если вы не подключаетесь к компьютеру по сети — это лишние процессы и лишняя дыра в защите.",
			apply:     func() { svcDisable(svcRemote, 4) },
			revert:    func() { svcRestore(svcRemote) },
			isApplied: func() bool { return svcAllAtLeast(svcRemote, 4) },
		},
		{ID: "svcphone", Group: svcGrp, Name: "Отключить «Связь с телефоном» и связанные устройства",
			Desc: "Платформа связанных устройств, карты, геолокация, кошелёк и демо-режим магазина. Перестанут работать Phone Link, «Обмен с устройствами поблизости» и синхронизация буфера обмена между ПК.",
			apply:     func() { svcDisable(svcPhoneLink, 4) },
			revert:    func() { svcRestore(svcPhoneLink) },
			isApplied: func() bool { return svcAllAtLeast(svcPhoneLink, 4) },
		},
		{ID: "svcxbox", Group: svcGrp, Name: "Отключить сетевые службы Xbox",
			Desc: "Вход в Xbox Live, облачные сохранения и сеть Xbox. Отключайте, только если не играете в игры из Game Pass / Microsoft Store. Геймпады Xbox продолжат работать.",
			apply:     func() { svcDisable(svcXbox, 4) },
			revert:    func() { svcRestore(svcXbox) },
			isApplied: func() bool { return svcAllAtLeast(svcXbox, 4) },
		},
		{ID: "svcprint", Group: svcGrp, Name: "Отключить печать и факс",
			Desc: "Диспетчер печати постоянно висит в фоне. Если принтера нет — смело отключайте. Появится принтер — откатите этот пункт.",
			apply:     func() { svcDisable(svcPrint, 4) },
			revert:    func() { svcRestore(svcPrint) },
			isApplied: func() bool { return svcAllAtLeast(svcPrint, 4) },
		},
		{ID: "svcsearch", Group: svcGrp, Name: "Отключить индексацию поиска Windows",
			Desc: "Индексатор постоянно сканирует файлы и нагружает диск. После отключения поиск в Пуске и Проводнике по-прежнему работает, но по содержимому файлов ищет медленнее.",
			apply:     func() { svcDisable(svcSearch, 4) },
			revert:    func() { svcRestore(svcSearch) },
			isApplied: func() bool { return svcAllAtLeast(svcSearch, 4) },
		},
	}
}
