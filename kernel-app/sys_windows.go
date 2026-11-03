package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

/* ============ Журнал ============ */
var (
	logMu    sync.Mutex
	logBuf   []LogLine
	logPath  string
	kernDir  string
)
type LogLine struct {
	Ts  string `json:"ts"`
	Lv  string `json:"lv"`
	Msg string `json:"msg"`
}
func initLog() {
	kernDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "KERNEL")
	os.MkdirAll(kernDir, 0755)
	logPath = filepath.Join(kernDir, "journal.log")
	ver := "Windows"
	if o := psLine(`(Get-CimInstance Win32_OperatingSystem).Caption`); o != "" {
		ver = o
	}
	logLine("INFO", "KERNEL запущен · "+ver)
}
func logLine(lv, msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	l := LogLine{Ts: time.Now().Format("15:04:05"), Lv: lv, Msg: msg}
	logBuf = append(logBuf, l)
	if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "[%s] %s %s\r\n", l.Ts, l.Lv, l.Msg)
		f.Close()
	}
}

/* ============ Запуск команд без окон ============ */
func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := cmd.CombinedOutput()
	return string(out), err
}
func ps(script string) (string, error) {
	// Принудительно UTF-8 на вывод, иначе кириллица (напр. «Майкрософт…») приходит мусором
	wrapped := "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8;$OutputEncoding=[System.Text.Encoding]::UTF8;" + script
	return run("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", wrapped)
}
func psLine(script string) string {
	o, _ := ps(script)
	return strings.TrimSpace(o)
}
func reg(args ...string) error {
	_, err := run("reg.exe", args...)
	return err
}

/* ============ Главная: сведения о системе ============ */
var sysInfoCache map[string]string // железо/ОС не меняются — кэшируем после первого запроса

func apiSysInfo() map[string]interface{} {
	if sysInfoCache != nil {
		return map[string]interface{}{
			"score": computeScore(), // оценка пересчитывается всегда (быстро, нативно)
			"cpu":   sysInfoCache["cpu"], "gpu": sysInfoCache["gpu"],
			"ram": sysInfoCache["ram"], "os": sysInfoCache["os"], "build": sysInfoCache["build"],
		}
	}
	type raw struct {
		CPU, GPU, OS, Build string
		RAM                 float64
	}
	out := psLine(`
$ci=Get-CimInstance Win32_Processor | Select-Object -First 1
# Все реальные видеокарты (встроенная + дискретная), без виртуальных адаптеров
$gpus=Get-CimInstance Win32_VideoController |
  Where-Object { $_.Name -and $_.PNPDeviceID -like 'PCI*' -and $_.Name -notmatch 'Basic|Remote|Virtual|Parsec|Mirror|DameWare|Meta' } |
  Select-Object -ExpandProperty Name -Unique
if(-not $gpus){ $gpus=(Get-CimInstance Win32_VideoController | Select-Object -First 1).Name }
$gpu=($gpus -join ' + ')
$os=Get-CimInstance Win32_OperatingSystem
# Реально установленная память (planки), а не доступная системе
$cap=(Get-CimInstance Win32_PhysicalMemory | Measure-Object -Property Capacity -Sum).Sum
if(-not $cap){ $cap=(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory }
$ram=[math]::Round($cap/1GB)
$b=(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').DisplayVersion
$caption=$os.Caption -replace 'Microsoft ','' -replace 'Майкрософт ',''
[pscustomobject]@{CPU=$ci.Name.Trim();GPU=$gpu;OS=$caption.Trim();Build=$b;RAM=$ram} | ConvertTo-Json -Compress`)
	var r raw
	json.Unmarshal([]byte(out), &r)
	if r.CPU == "" {
		r.CPU = "—"
	}
	sysInfoCache = map[string]string{
		"cpu": r.CPU, "gpu": r.GPU, "ram": fmt.Sprintf("%.0f ГБ", r.RAM), "os": r.OS, "build": r.Build,
	}
	return map[string]interface{}{
		"score": computeScore(), "cpu": r.CPU, "gpu": r.GPU,
		"ram": fmt.Sprintf("%.0f ГБ", r.RAM), "os": r.OS, "build": r.Build,
	}
}
// Оценка «состояния»: сколько рекомендуемых твиков ещё НЕ применено
func computeScore() int {
	ensureTweaks()
	applied := 0
	for _, t := range tweakDefs {
		if t.Recommended && t.isApplied() {
			applied++
		}
	}
	total := 0
	for _, t := range tweakDefs {
		if t.Recommended {
			total++
		}
	}
	if total == 0 {
		return 100
	}
	return 40 + int(float64(applied)/float64(total)*60)
}

/* ============ Ускорение: твики ============ */
type Tweak struct {
	ID, Group, Name, Desc string
	Recommended, Reboot   bool
	Details               string // необязательный развёрнутый текст для «Подробнее»
	VBS                   bool   // особый твик VBS: статус + предупреждение + отдельный откат
	apply                 func()
	revert                func()
	isApplied             func() bool
}
func regDword(root, path, name string, val int) {
	// нативная запись; при ошибке (например, нет прав на ветку) — запасной reg.exe
	if err := regWriteDwordNative(root, path, name, uint32(val)); err != nil {
		reg("add", root+`\`+path, "/v", name, "/t", "REG_DWORD", "/d", fmt.Sprint(val), "/f")
	}
}
func regDel(root, path, name string) { regDeleteValueNative(root, path, name) }
func dwordIs(root, path, name string, val int) bool {
	v, ok := regReadDword(root, path, name)
	return ok && int(v) == val
}

/* ---- Схема питания ---- */
// Своя копия «Максимальной производительности» с фиксированным GUID: powercfg -duplicatescheme
// без второго аргумента каждый раз создаёт новую копию со случайным GUID, а сам шаблон
// e9a42b02 обычно нельзя сделать активным — поэтому раньше твик плодил копии и не включался.
const (
	ultimateTemplate = "e9a42b02-d5df-448d-aa00-03f14749eb61"
	kernelScheme     = "4b3e1c6a-7e1d-4a6f-9c2b-6b5e4b2a9f01"
	balancedScheme   = "381b4222-f694-41f0-9685-ff5bb260df2e"
)

func powerApply() {
	if !regKeyExists("HKLM", powerSchemesPath+`\`+kernelScheme) {
		run("powercfg.exe", "-duplicatescheme", ultimateTemplate, kernelScheme)
	}
	run("powercfg.exe", "-setactive", kernelScheme)
	cleanupUltimateCopies()
}

const powerSchemesPath = `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`

func activePowerScheme() string {
	s, _ := regReadString("HKLM", powerSchemesPath, "ActivePowerScheme")
	return strings.ToLower(s)
}

// Удаляем лишние копии «Максимальной производительности», которые наплодили старые версии KERNEL.
// Ищем по реестру: имя копии хранится как ссылка на строку powrprof.dll,-19 — не зависит от языка Windows.
func cleanupUltimateCopies() {
	active := activePowerScheme()
	removed := 0
	for _, g := range regSubKeys("HKLM", powerSchemesPath) {
		g = strings.ToLower(g)
		if g == kernelScheme || g == active || g == ultimateTemplate { // шаблон не трогаем — из него создаётся схема
			continue
		}
		if name, _ := regReadString("HKLM", powerSchemesPath+`\`+g, "FriendlyName"); strings.Contains(name, "powrprof.dll,-19,") {
			if _, err := run("powercfg.exe", "/delete", g); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		logLine("OK", fmt.Sprintf("Удалено лишних копий схемы питания: %d", removed))
	}
}

func powerRevert() {
	run("powercfg.exe", "-setactive", balancedScheme)
	run("powercfg.exe", "/delete", kernelScheme)
}

// Есть ли у ПК батарея (ноутбук) — там «Максимальная производительность» только греет и сажает заряд
func hasBattery() bool {
	var st struct {
		ACLineStatus, BatteryFlag, BatteryLifePercent, SystemStatusFlag byte
		BatteryLifeTime, BatteryFullLifeTime                            uint32
	}
	r, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemPowerStatus").Call(uintptr(unsafe.Pointer(&st)))
	return r != 0 && st.BatteryFlag != 128 && st.BatteryFlag != 255
}

/* ---- Оптимизация оконных игр (flip model) ---- */
const dxPrefsPath = `SOFTWARE\Microsoft\DirectX\UserGpuPreferences`

// Меняем/добавляем одну настройку в строке DirectXUserGlobalSettings, не трогая остальные (VRR и т.п.)
func dxGlobalSet(key, val string) {
	cur, _ := regReadString("HKCU", dxPrefsPath, "DirectXUserGlobalSettings")
	parts := []string{}
	for _, p := range strings.Split(cur, ";") {
		if p = strings.TrimSpace(p); p != "" && !strings.HasPrefix(p, key+"=") {
			parts = append(parts, p)
		}
	}
	if val != "" {
		parts = append(parts, key+"="+val)
	}
	s := ""
	for _, p := range parts {
		s += p + ";"
	}
	reg("add", `HKCU\`+dxPrefsPath, "/v", "DirectXUserGlobalSettings", "/t", "REG_SZ", "/d", s, "/f")
}
func dxGlobalHas(kv string) bool {
	cur, _ := regReadString("HKCU", dxPrefsPath, "DirectXUserGlobalSettings")
	return strings.Contains(cur, kv)
}

var (
	tweakDefs []Tweak
	tweakOnce sync.Once
)

// Твики создаются один раз; вызывается из любого потока (прогрев на старте и вызовы из JS идут параллельно)
func ensureTweaks() { tweakOnce.Do(initTweaks) }

func initTweaks() {
	tweakDefs = []Tweak{
		{ID: "power", Group: "Производительность и FPS", Name: "Схема питания «Максимальная производительность»",
			Desc:        "Ядра процессора не «засыпают» и не сбрасывают частоту между кадрами — меньше микрозадержек и ровнее 1% low. Средний FPS на настольном ПК растёт мало (0–3%). На ноутбуке не рекомендуется: греет и быстро сажает батарею.",
			Recommended: !hasBattery(),
			apply:       powerApply,
			revert:      powerRevert,
			isApplied:   func() bool { return activePowerScheme() == kernelScheme },
		},
		{ID: "flipmodel", Group: "Производительность и FPS", Name: "Оптимизация игр в окне и без рамки",
			Desc:        "Штатная функция Windows 11: старые DirectX 10/11 игры в оконном и «полноэкранном без рамки» режиме выводят кадры напрямую, как в полноэкранном. Ниже задержка ввода, работает G-Sync/FreeSync. Действует со следующего запуска игры.",
			Recommended: true,
			apply:       func() { dxGlobalSet("SwapEffectUpgradeEnable", "1") },
			revert:      func() { dxGlobalSet("SwapEffectUpgradeEnable", "") },
			isApplied:   func() bool { return dxGlobalHas("SwapEffectUpgradeEnable=1") },
		},
		{ID: "vrr", Group: "Производительность и FPS", Name: "G-Sync / FreeSync для всех игр",
			Desc:        "Включает переменную частоту обновления и в тех DirectX 11/12 играх, которые сами её не поддерживают. Нет разрывов кадра без V-Sync и его задержки. Работает, если монитор поддерживает G-Sync/FreeSync; на остальных ничего не меняет.",
			Recommended: true,
			apply:       func() { dxGlobalSet("VRROptimizeEnable", "1") },
			revert:      func() { dxGlobalSet("VRROptimizeEnable", "") },
			isApplied:   func() bool { return dxGlobalHas("VRROptimizeEnable=1") },
		},
		{ID: "throttle", Group: "Производительность и FPS", Name: "Отключить энергосбережение фоновых программ",
			Desc: "Windows урезает частоту процессора для свёрнутых программ (EcoQoS). Если во время игры в фоне идёт стрим (OBS), Discord или запись — они перестанут подтормаживать и не будут «дёргать» игру. Без фоновых программ эффекта нет. На ноутбуке чуть сильнее тратится батарея.", Reboot: true,
			apply:     func() { regDword("HKLM", `SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`, "PowerThrottlingOff", 1) },
			revert:    func() { regDel("HKLM", `SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`, "PowerThrottlingOff") },
			isApplied: func() bool { return dwordIs("HKLM", `SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`, "PowerThrottlingOff", 1) },
		},
		{ID: "gamedvr", Group: "Производительность и FPS", Name: "Отключить запись игр (Game DVR)",
			Desc:        "Выключает фоновую запись Xbox Game Bar. Если запись «последних 30 секунд» была включена, она постоянно кодирует видео и отнимает несколько процентов FPS. Если не была — прироста не будет, но и вреда нет.",
			Recommended: true,
			apply: func() {
				regDword("HKCU", `System\GameConfigStore`, "GameDVR_Enabled", 0)
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, "AppCaptureEnabled", 0)
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, "HistoricalCaptureEnabled", 0)
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Windows\GameDVR`, "AllowGameDVR", 0)
			},
			revert: func() {
				regDword("HKCU", `System\GameConfigStore`, "GameDVR_Enabled", 1)
				regDel("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, "AppCaptureEnabled")
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Windows\GameDVR`, "AllowGameDVR")
			},
			isApplied: func() bool { return dwordIs("HKCU", `System\GameConfigStore`, "GameDVR_Enabled", 0) },
		},
		{ID: "hags", Group: "Производительность и FPS", Name: "Аппаратное ускорение планирования GPU (HAGS)",
			Desc: "Видеокарта сама планирует свою очередь кадров. Обязательно для генерации кадров DLSS (RTX 40/50) — без HAGS она не включится. На старых видеокартах и драйверах иногда даёт фризы, поэтому по умолчанию не трогаем.", Recommended: false, Reboot: true,
			apply:     func() { regDword("HKLM", `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, "HwSchMode", 2) },
			revert:    func() { regDword("HKLM", `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, "HwSchMode", 1) },
			isApplied: func() bool { return dwordIs("HKLM", `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, "HwSchMode", 2) },
		},
		{ID: "gamemode", Group: "Производительность и FPS", Name: "Игровой режим Windows",
			Desc: "Windows отдаёт приоритет активной игре и не ставит обновления во время игры. В Windows 10/11 он включён по умолчанию — твик просто гарантирует, что его никто не выключил.", Recommended: true,
			apply: func() {
				regDword("HKCU", `SOFTWARE\Microsoft\GameBar`, "AllowAutoGameMode", 1)
				regDword("HKCU", `SOFTWARE\Microsoft\GameBar`, "AutoGameModeEnabled", 1)
				// подчищаем вредный твик из старых версий, если он был применён
				p := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
				if v, ok := regReadDword("HKLM", p, "SystemResponsiveness"); ok && v == 0 {
					regDword("HKLM", p, "SystemResponsiveness", 20)
				}
			},
			revert: func() {
				regDword("HKCU", `SOFTWARE\Microsoft\GameBar`, "AutoGameModeEnabled", 0)
				// возвращаем значения по умолчанию, убираем вредные твики старых версий
				p := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
				regDword("HKLM", p, "SystemResponsiveness", 20)
				regDel("HKLM", p+`\Tasks\Games`, "GPU Priority")
				regDel("HKLM", p+`\Tasks\Games`, "Priority")
			},
			isApplied: func() bool { return dwordIs("HKCU", `SOFTWARE\Microsoft\GameBar`, "AutoGameModeEnabled", 1) },
		},
		{ID: "bgapps", Group: "Фоновые процессы", Name: "Запретить работу приложений в фоне",
			Desc: "Встроенные приложения из Microsoft Store (Почта, Погода, Phone Link и др.) не висят в фоне, пока закрыты. Экономит немного памяти и процессов. Минус: от этих приложений перестанут приходить уведомления.", Recommended: true,
			apply:     func() { regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, "GlobalUserDisabled", 1) },
			revert:    func() { regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, "GlobalUserDisabled", 0) },
			isApplied: func() bool { return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, "GlobalUserDisabled", 1) },
		},
		{ID: "telemetry", Group: "Фоновые процессы", Name: "Отключить телеметрию и сбор данных",
			Desc: "Останавливает службу DiagTrack, которая собирает и отправляет диагностику в Microsoft. Минус один фоновый процесс и меньше периодической нагрузки на диск и сеть.", Recommended: true,
			apply: func() {
				run("sc.exe", "stop", "DiagTrack")
				run("sc.exe", "config", "DiagTrack", "start=", "disabled")
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "AllowTelemetry", 0)
			},
			revert: func() {
				run("sc.exe", "config", "DiagTrack", "start=", "auto")
				run("sc.exe", "start", "DiagTrack")
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "AllowTelemetry")
			},
			isApplied: func() bool { return serviceStart("DiagTrack") == 4 },
		},
		{ID: "sysmain", Group: "Фоновые процессы", Name: "Отключить SysMain (Superfetch)",
			Desc: "Служба заранее подгружает в память часто используемые программы. На SSD выигрыш от неё мал, но вместе с ней отключается и сжатие памяти — при нехватке ОЗУ (8–16 ГБ) станет хуже. Имеет смысл только при 32 ГБ и больше.",
			apply: func() {
				run("sc.exe", "stop", "SysMain")
				run("sc.exe", "config", "SysMain", "start=", "disabled")
			},
			revert: func() {
				run("sc.exe", "config", "SysMain", "start=", "auto")
				run("sc.exe", "start", "SysMain")
			},
			isApplied: func() bool { return serviceStart("SysMain") == 4 },
		},
		{ID: "tips", Group: "Интерфейс", Name: "Убрать рекламу и «советы» Windows",
			Desc: "Отключает предложения в меню Пуск, подсказки, рекламные уведомления и тихую установку «рекомендованных» приложений.", Recommended: true,
			apply: func() {
				c := `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`
				for _, v := range []string{"SubscribedContent-338388Enabled", "SubscribedContent-338389Enabled", "SystemPaneSuggestionsEnabled", "SilentInstalledAppsEnabled"} {
					regDword("HKCU", c, v, 0)
				}
			},
			revert: func() {
				c := `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`
				for _, v := range []string{"SubscribedContent-338388Enabled", "SubscribedContent-338389Enabled", "SystemPaneSuggestionsEnabled", "SilentInstalledAppsEnabled"} {
					regDword("HKCU", c, v, 1)
				}
			},
			isApplied: func() bool { return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`, "SystemPaneSuggestionsEnabled", 0) },
		},
		{ID: "mouseaccel", Group: "Производительность и FPS", Name: "Отключить акселерацию мыши",
			Desc: "Убирает «улучшение точности указателя». На FPS не влияет, но прицел становится предсказуемым — движение мыши 1:1, проще набить мышечную память.", Recommended: true,
			apply: func() {
				m := `Control Panel\Mouse`
				reg("add", `HKCU\`+m, "/v", "MouseSpeed", "/t", "REG_SZ", "/d", "0", "/f")
				reg("add", `HKCU\`+m, "/v", "MouseThreshold1", "/t", "REG_SZ", "/d", "0", "/f")
				reg("add", `HKCU\`+m, "/v", "MouseThreshold2", "/t", "REG_SZ", "/d", "0", "/f")
			},
			revert: func() {
				m := `Control Panel\Mouse`
				reg("add", `HKCU\`+m, "/v", "MouseSpeed", "/t", "REG_SZ", "/d", "1", "/f")
				reg("add", `HKCU\`+m, "/v", "MouseThreshold1", "/t", "REG_SZ", "/d", "6", "/f")
				reg("add", `HKCU\`+m, "/v", "MouseThreshold2", "/t", "REG_SZ", "/d", "10", "/f")
			},
			isApplied: func() bool { v, _ := regReadString("HKCU", `Control Panel\Mouse`, "MouseSpeed"); return v == "0" },
		},
		{ID: "visualfx", Group: "Интерфейс", Name: "Режим «максимум производительности» для эффектов",
			Desc: "Отключает анимации, тени и прозрачность окон. Заметно только на слабых ПК со встроенной графикой — на FPS в играх не влияет. Вступает в силу после выхода из системы.",
			apply: func() {
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, "VisualFXSetting", 2)
				reg("add", `HKCU\Control Panel\Desktop`, "/v", "UserPreferencesMask", "/t", "REG_BINARY", "/d", "9012038010000000", "/f")
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\DWM`, "EnableAeroPeek", 0)
			},
			revert: func() {
				// значения Windows 10/11 по умолчанию («на усмотрение Windows»)
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, "VisualFXSetting", 0)
				reg("add", `HKCU\Control Panel\Desktop`, "/v", "UserPreferencesMask", "/t", "REG_BINARY", "/d", "9E1E078012000000", "/f")
				regDword("HKCU", `SOFTWARE\Microsoft\Windows\DWM`, "EnableAeroPeek", 1)
			},
			isApplied: func() bool { return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, "VisualFXSetting", 2) },
		},
		{ID: "fastmenu", Group: "Интерфейс", Name: "Ускорить отклик меню и окон",
			Desc: "Убирает задержку 0,4 с перед появлением подменю и анимацию сворачивания окон. Рабочий стол ощущается отзывчивее. Вступает в силу после выхода из системы.",
			apply: func() {
				reg("add", `HKCU\Control Panel\Desktop`, "/v", "MenuShowDelay", "/t", "REG_SZ", "/d", "0", "/f")
				reg("add", `HKCU\Control Panel\Desktop\WindowMetrics`, "/v", "MinAnimate", "/t", "REG_SZ", "/d", "0", "/f")
			},
			revert: func() {
				reg("add", `HKCU\Control Panel\Desktop`, "/v", "MenuShowDelay", "/t", "REG_SZ", "/d", "400", "/f")
				reg("add", `HKCU\Control Panel\Desktop\WindowMetrics`, "/v", "MinAnimate", "/t", "REG_SZ", "/d", "1", "/f")
			},
			isApplied: func() bool { v, _ := regReadString("HKCU", `Control Panel\Desktop`, "MenuShowDelay"); return v == "0" },
		},
		{ID: "nagle", Group: "Производительность и FPS", Name: "Отключить задержку TCP-подтверждений (Nagle)",
			Desc: "Помогает только старым онлайн-играм на TCP (WoW, Lineage, часть MMO): меньше рывков при задержках сети. Современные шутеры (CS2, Valorant, Fortnite, Apex) работают по UDP — на них не влияет. Нужна перезагрузка.", Reboot: true,
			apply: func() {
				ps(`$base='HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces'
Get-ChildItem $base | ForEach-Object {
  New-ItemProperty -Path $_.PSPath -Name 'TcpAckFrequency' -Value 1 -PropertyType DWord -Force | Out-Null
  New-ItemProperty -Path $_.PSPath -Name 'TCPNoDelay' -Value 1 -PropertyType DWord -Force | Out-Null
}`)
			},
			revert: func() {
				ps(`$base='HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces'
Get-ChildItem $base | ForEach-Object { Remove-ItemProperty -Path $_.PSPath -Name 'TcpAckFrequency','TCPNoDelay' -ErrorAction SilentlyContinue }`)
			},
			isApplied: func() bool {
				base := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`
				for _, sk := range regSubKeys("HKLM", base) {
					if v, ok := regReadDword("HKLM", base+`\`+sk, "TcpAckFrequency"); ok && v == 1 {
						return true
					}
				}
				return false
			},
		},
		{ID: "vbs", Group: "Производительность и FPS", Name: "Отключить VBS (виртуализация безопасности)",
			Desc: "Самый заметный твик для игр: в части игр +5–10% FPS и ровнее 1% low. Microsoft сама советует его геймерам. Взамен — слабее защита ядра.", Reboot: true, VBS: true,
			Details: "VBS (Virtualization-Based Security) — функция защиты Windows, которая запускает часть ядра системы в отдельной виртуальной машине. Она защищает от вредоносных драйверов и кражи паролей, но постоянно работающий гипервизор добавляет задержку при работе с памятью и процессором.\n\nЧто даст отключение:\n• ниже задержка памяти (обычно на 3–7 нс в AIDA64);\n• прирост FPS в части игр, сильнее всего — в 1% low (меньше просадок);\n• чуть меньше нагрузки на систему в фоне.\n\nЧем вы жертвуете:\n• защитой ядра от вредоносных драйверов и некоторых видов атак;\n• перестанут работать WSL2, Hyper-V, Windows Sandbox и Android-эмуляторы на базе Hyper-V.\n\nИзменения вступают в силу после перезагрузки и полностью откатываются кнопкой «Вернуть как было».",
			apply:     vbsApply,
			revert:    vbsRevert,
			isApplied: vbsTweakApplied,
		},
		{ID: "reserved", Group: "Место на диске", Name: "Освободить ~7 ГБ «зарезервированного хранилища»",
			Desc: "Windows держит около 7 ГБ диска «про запас» под обновления. Отключаем — место возвращается. На скорость не влияет. Если сейчас ставится обновление, место освободится после его завершения.",
			apply: func() {
				run("DISM.exe", "/Online", "/Set-ReservedStorageState", "/State:Disabled", "/NoRestart")
				regDword("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\ReserveManager`, "ShippedWithReserves", 0)
			},
			revert: func() {
				run("DISM.exe", "/Online", "/Set-ReservedStorageState", "/State:Enabled", "/NoRestart")
				regDword("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\ReserveManager`, "ShippedWithReserves", 1)
			},
			isApplied: func() bool { return dwordIs("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\ReserveManager`, "ShippedWithReserves", 0) },
		},
		{ID: "hibernate", Group: "Место на диске", Name: "Отключить гибернацию",
			Desc: "Удаляет файл hiberfil.sys (обычно 40% от объёма ОЗУ). Вместе с ним отключается «Быстрый запуск» — включение с нуля станет на пару секунд дольше, зато каждый запуск «чистый». Сон работает как раньше.",
			apply:     func() { run("powercfg.exe", "/h", "off") },
			revert:    func() { run("powercfg.exe", "/h", "on") },
			isApplied: func() bool { _, err := os.Stat(os.Getenv("SystemDrive") + `\hiberfil.sys`); return err != nil },
		},
	}
	tweakDefs = append(tweakDefs, backgroundTweaks()...)

	// Порядок разделов на вкладке «Ускорение»
	order := map[string]int{"Производительность и FPS": 0, "Фоновые процессы": 1, "Фоновые службы (по выбору)": 2, "Интерфейс": 3, "Место на диске": 4}
	sort.SliceStable(tweakDefs, func(i, j int) bool { return order[tweakDefs[i].Group] < order[tweakDefs[j].Group] })
}
func (t Tweak) toMap() map[string]interface{} {
	return map[string]interface{}{
		"id": t.ID, "group": t.Group, "name": t.Name, "desc": t.Desc,
		"on": t.Recommended, "applied": t.isApplied(), "reboot": t.Reboot,
		"details": t.Details, "vbs": t.VBS,
	}
}
func apiTweaks() []map[string]interface{} {
	ensureTweaks()
	res := []map[string]interface{}{}
	for _, t := range tweakDefs {
		res = append(res, t.toMap())
	}
	return res
}
func findTweak(id string) *Tweak {
	for i := range tweakDefs {
		if tweakDefs[i].ID == id {
			return &tweakDefs[i]
		}
	}
	return nil
}
func apiApplyTweaks(ids []string) bool {
	ensureTweaks()
	for _, id := range ids {
		if t := findTweak(id); t != nil {
			safe(t.Name, t.apply)
		}
	}
	return true
}
func apiApplyAll() bool {
	ensureTweaks()
	logLine("INFO", "=== Автонастройка: старт ===")
	createRestorePoint()
	for i := range tweakDefs {
		if tweakDefs[i].Recommended {
			safe(tweakDefs[i].Name, tweakDefs[i].apply)
		}
	}
	safe("Очистка временных файлов", func() { cleanTemp() })
	logLine("OK", "=== Автонастройка завершена ===")
	return true
}
func apiRevertAll() bool {
	ensureTweaks()
	logLine("INFO", "=== Откат изменений ===")
	for i := range tweakDefs {
		if tweakDefs[i].revert != nil {
			safe("Откат: "+tweakDefs[i].Name, tweakDefs[i].revert)
		}
	}
	return true
}
func safe(what string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			logLine("ERR", fmt.Sprintf("%s — ошибка: %v", what, r))
		}
	}()
	f()
	logLine("OK", what)
}
func createRestorePoint() {
	safe("Точка восстановления системы", func() {
		ps(`Enable-ComputerRestore -Drive "$env:SystemDrive\"; Checkpoint-Computer -Description "KERNEL" -RestorePointType MODIFY_SETTINGS`)
	})
}

/* ============ Очистка ============ */
type CleanItem struct {
	ID, Name, Desc string
	Bytes          int64
	On             bool
	paths          []string
}
func cleanDefs() []CleanItem {
	tmp := os.Getenv("TEMP")
	win := os.Getenv("WINDIR")
	la := os.Getenv("LOCALAPPDATA")
	pd := os.Getenv("ProgramData")
	return []CleanItem{
		{ID: "temp", Name: "Временные файлы", Desc: "Папки TEMP пользователя и Windows — главный источник мусора.", On: true, paths: []string{tmp, win + `\Temp`}},
		{ID: "update", Name: "Кэш Windows Update", Desc: "Скачанные установщики обновлений, которые уже применены.", On: true, paths: []string{win + `\SoftwareDistribution\Download`}},
		{ID: "thumbs", Name: "Кэш эскизов и иконок", Desc: "Миниатюры и значки — пересоздадутся сами.", On: true, paths: []string{la + `\Microsoft\Windows\Explorer`}},
		{ID: "inetcache", Name: "Кэш интернета Windows", Desc: "Временные файлы встроенных веб-компонентов.", On: true, paths: []string{la + `\Microsoft\Windows\INetCache`}},
		{ID: "shadercache", Name: "Кэш шейдеров видеокарты", Desc: "Кэш DirectX/NVIDIA/AMD. Пересоберётся в играх — иногда убирает фризы компиляции шейдеров.", On: true, paths: []string{la + `\D3DSCache`, la + `\NVIDIA\DXCache`, la + `\NVIDIA\GLCache`, la + `\AMD\DxCache`}},
		{ID: "crashdumps", Name: "Отчёты об ошибках и дампы", Desc: "Дампы памяти после сбоев и очереди отчётов — весят много, не нужны.", On: true, paths: []string{la + `\CrashDumps`, win + `\Minidump`, pd + `\Microsoft\Windows\WER\ReportQueue`, pd + `\Microsoft\Windows\WER\ReportArchive`}},
		{ID: "delivery", Name: "Кэш раздачи обновлений", Desc: "Файлы, которые Windows раздаёт другим ПК в сети.", On: true, paths: []string{win + `\SoftwareDistribution\DeliveryOptimization`}},
		{ID: "prefetch", Name: "Prefetch", Desc: "Файлы предзагрузки, Windows пересоздаст их.", On: true, paths: []string{win + `\Prefetch`}},
		{ID: "logs", Name: "Журналы системы", Desc: "Старые лог-файлы Windows.", On: true, paths: []string{win + `\Logs`, pd + `\Microsoft\Windows\WER\Temp`}},
		{ID: "recycle", Name: "Корзина", Desc: "Удалённые файлы на всех дисках.", paths: []string{}},
	}
}
func dirSize(path string) int64 {
	var total int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}
func apiCleanScan() []map[string]interface{} {
	res := []map[string]interface{}{}
	for _, c := range cleanDefs() {
		var sz int64
		for _, p := range c.paths {
			sz += dirSize(p)
		}
		res = append(res, map[string]interface{}{"id": c.ID, "name": c.Name, "desc": c.Desc, "bytes": sz, "on": c.On})
	}
	return res
}
func cleanTemp() {
	for _, p := range []string{os.Getenv("TEMP"), os.Getenv("WINDIR") + `\Temp`, os.Getenv("WINDIR") + `\Prefetch`} {
		clearDir(p)
	}
}
func clearDir(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, e := range entries {
		os.RemoveAll(filepath.Join(path, e.Name()))
	}
}
func apiClean(ids []string) bool {
	defs := map[string]CleanItem{}
	for _, c := range cleanDefs() {
		defs[c.ID] = c
	}
	logLine("INFO", "=== Очистка ===")
	for _, id := range ids {
		c, ok := defs[id]
		if !ok {
			continue
		}
		if id == "recycle" {
			safe("Очистка корзины", func() { ps(`Clear-RecycleBin -Force -ErrorAction SilentlyContinue`) })
			continue
		}
		var before int64
		for _, p := range c.paths {
			before += dirSize(p)
			clearDir(p)
		}
		logLine("OK", fmt.Sprintf("%s — освобождено %s", c.Name, fmtBytes(before)))
	}
	return true
}
func fmtBytes(b int64) string {
	switch {
	case b > 1<<30:
		return fmt.Sprintf("%.1f ГБ", float64(b)/(1<<30))
	case b > 1<<20:
		return fmt.Sprintf("%d МБ", b/(1<<20))
	default:
		return fmt.Sprintf("%d КБ", b/(1<<10))
	}
}

/* ============ UWP ============ */
// Каталог встроенных приложений: friendly RU имя -> шаблон имени Appx-пакета
var uwpCatalog = []struct {
	Name, Pkg string
	Junk      bool
}{
	{"Копилот", "Microsoft.Copilot", true}, {"Copilot (365)", "Microsoft.MicrosoftOfficeHub", true},
	{"Новости", "Microsoft.BingNews", true}, {"Погода", "Microsoft.BingWeather", true},
	{"Bing Поиск", "Microsoft.BingSearch", true}, {"Clipchamp", "Clipchamp.Clipchamp", true},
	{"Xbox", "Microsoft.GamingApp", true}, {"Xbox TCUI", "Microsoft.Xbox.TCUI", true},
	{"Xbox Game Bar", "Microsoft.XboxGamingOverlay", true}, {"Xbox Speech", "Microsoft.XboxSpeechToTextOverlay", true},
	{"Xbox Identity", "Microsoft.XboxIdentityProvider", true}, {"Solitaire Collection", "Microsoft.MicrosoftSolitaireCollection", true},
	{"Советы", "Microsoft.Getstarted", true}, {"Начало работы", "Microsoft.GetHelp", true},
	{"Люди", "Microsoft.People", true}, {"Your Phone / Связь с телефоном", "Microsoft.YourPhone", true},
	{"Groove Музыка", "Microsoft.ZuneMusic", true}, {"Кино и ТВ", "Microsoft.ZuneVideo", true},
	{"Office Hub", "Microsoft.MicrosoftOfficeHub", true}, {"Teams", "MicrosoftTeams", true},
	{"To Do", "Microsoft.Todos", true}, {"Power Automate", "Microsoft.PowerAutomateDesktop", true},
	{"Обратная связь", "Microsoft.WindowsFeedbackHub", true}, {"Карты", "Microsoft.WindowsMaps", true},
	{"OneNote", "Microsoft.Office.OneNote", true}, {"Outlook (новый)", "Microsoft.OutlookForWindows", true},
	{"Dev Home", "Microsoft.Windows.DevHome", true}, {"Быстрая поддержка", "MicrosoftCorporationII.QuickAssist", true},
	{"Cross Device", "MicrosoftWindows.CrossDevice", true}, {"Family", "MicrosoftCorporationII.MicrosoftFamily", true},
	// Спец-пункты (не Appx): id начинается с "__", удаляются через реестр
	{"Значок «Сведения об изображении» (Spotlight)", "__spotlight", true},
	{"Виджеты (погода/новости на панели)", "__widgets", true},
	// Полезные — по умолчанию НЕ отмечены
	{"Калькулятор", "Microsoft.WindowsCalculator", false}, {"Камера", "Microsoft.WindowsCamera", false},
	{"Фотографии", "Microsoft.Windows.Photos", false}, {"Ножницы", "Microsoft.ScreenSketch", false},
	{"Запись голоса", "Microsoft.WindowsSoundRecorder", false}, {"Будильники и часы", "Microsoft.WindowsAlarms", false},
	{"Терминал", "Microsoft.WindowsTerminal", false}, {"Блокнот", "Microsoft.WindowsNotepad", false},
	{"Paint", "Microsoft.Paint", false}, {"Медиаплеер", "Microsoft.ZuneMusic", false},
}

var installedPkgs map[string]bool

func refreshInstalled() {
	installedPkgs = map[string]bool{}
	out := psLine(`(Get-AppxPackage | Select-Object -ExpandProperty Name) -join "|"`)
	for _, n := range strings.Split(out, "|") {
		installedPkgs[strings.ToLower(strings.TrimSpace(n))] = true
	}
}
func isInstalled(pkg string) bool {
	if installedPkgs == nil {
		refreshInstalled()
	}
	key := strings.ToLower(pkg)
	for name := range installedPkgs {
		if strings.Contains(name, key) {
			return true
		}
	}
	return false
}
// Спец-пункты: «установлен» = функция/значок ещё активны
func specialPresent(id string) bool {
	switch id {
	case "__spotlight":
		// значок на рабочем столе виден, если флаг скрытия != 1
		return !dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel`, "{2cc5ca98-6485-489a-920e-b3e88a6ccce3}", 1)
	case "__widgets":
		return !dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, "TaskbarDa", 0)
	}
	return false
}
func apiUwpList() []map[string]interface{} {
	refreshInstalled()
	res := []map[string]interface{}{}
	for _, a := range uwpCatalog {
		inst := false
		if strings.HasPrefix(a.Pkg, "__") {
			inst = specialPresent(a.Pkg)
		} else {
			inst = isInstalled(a.Pkg)
		}
		res = append(res, map[string]interface{}{
			"id": a.Pkg, "name": a.Name, "installed": inst, "junk": a.Junk, "sel": a.Junk && inst,
		})
	}
	return res
}

/* ============ Программы / браузеры ============ */
var appsCatalog = []map[string]interface{}{
	{"id": "npp", "name": "Notepad++", "cat": "Редактор", "letter": "N+", "bg": "#1E3A24", "fg": "#7CD67C", "desc": "Быстрый текстовый редактор с подсветкой кода. Замена Блокноту.", "offline": false, "winget": "Notepad++.Notepad++", "url": "https://notepad-plus-plus.org"},
	{"id": "mpc", "name": "MPC-HC", "cat": "Видео", "letter": "▶", "bg": "#1E2A3A", "fg": "#7AA7D9", "desc": "Лёгкий видеоплеер, играет почти всё. Замена «Кино и ТВ».", "offline": false, "winget": "clsid2.mpc-hc", "url": "https://github.com/clsid2/mpc-hc/releases"},
	{"id": "imageglass", "name": "ImageGlass 8.12", "cat": "Фото", "letter": "◆", "bg": "#3A2E1A", "fg": "#E8C15A", "desc": "Быстрый просмотр изображений. Замена «Фотографиям».", "offline": false, "winget": "DuongDieuPhap.ImageGlass", "url": "https://imageglass.org"},
	{"id": "raven", "name": "Raven", "cat": "Утилита", "letter": "✦", "bg": "#2A1E3A", "fg": "#B98AD9", "desc": "Установка приложений из Microsoft Store без самого Store.", "offline": false, "winget": "", "url": "https://github.com/mjishnu/Raven/releases"},
	{"id": "firefox", "name": "Mozilla Firefox", "cat": "Браузер", "letter": "F", "bg": "#3A2410", "fg": "#E8A15A", "desc": "Независимый браузер на своём движке.", "offline": false, "winget": "Mozilla.Firefox", "url": "https://www.mozilla.org/firefox"},
	{"id": "chrome", "name": "Google Chrome", "cat": "Браузер", "letter": "◉", "bg": "#2A2A2E", "fg": "#E8E8EE", "desc": "Самый популярный браузер на Chromium.", "offline": false, "winget": "Google.Chrome", "url": "https://www.google.com/chrome"},
	{"id": "brave", "name": "Brave", "cat": "Браузер", "letter": "B", "bg": "#3A1E10", "fg": "#E87A4A", "desc": "Chromium со встроенной блокировкой рекламы.", "offline": false, "winget": "Brave.Brave", "url": "https://brave.com/download"},
	{"id": "yandex", "name": "Yandex Browser", "cat": "Браузер", "letter": "Я", "bg": "#3A1010", "fg": "#E85C5C", "desc": "Chromium-браузер от Яндекса.", "offline": false, "winget": "Yandex.Browser", "url": "https://browser.yandex.ru"},
}
func installersDir() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "Installers")
}
// Ключевые слова для определения установленных программ в реестре
var appDetect = map[string]string{
	"npp": "Notepad++", "mpc": "MPC-HC", "imageglass": "ImageGlass", "raven": "Raven",
	"firefox": "Mozilla Firefox", "chrome": "Google Chrome", "brave": "Brave", "yandex": "Yandex",
}
var installedNames []string

func refreshInstalledPrograms() {
	out := psLine(`
$p=@('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
     'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
     'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*')
(Get-ItemProperty $p -ErrorAction SilentlyContinue | Select-Object -ExpandProperty DisplayName -ErrorAction SilentlyContinue) -join '|'`)
	installedNames = strings.Split(strings.ToLower(out), "|")
}
// Где реально лежит exe программы (проверяем файл, а не запись в реестре)
var appPaths = map[string][]string{
	"npp":        {`%ProgramFiles%\Notepad++\notepad++.exe`, `%ProgramFiles(x86)%\Notepad++\notepad++.exe`},
	"mpc":        {`%ProgramFiles%\MPC-HC\mpc-hc64.exe`, `%ProgramFiles(x86)%\MPC-HC\mpc-hc.exe`},
	"imageglass": {`%ProgramFiles%\ImageGlass\ImageGlass.exe`, `%ProgramFiles(x86)%\ImageGlass\ImageGlass.exe`},
	"firefox":    {`%ProgramFiles%\Mozilla Firefox\firefox.exe`, `%ProgramFiles(x86)%\Mozilla Firefox\firefox.exe`, `%LOCALAPPDATA%\Mozilla Firefox\firefox.exe`},
	"chrome":     {`%ProgramFiles%\Google\Chrome\Application\chrome.exe`, `%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe`, `%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe`},
	"brave":      {`%ProgramFiles%\BraveSoftware\Brave-Browser\Application\brave.exe`, `%ProgramFiles(x86)%\BraveSoftware\Brave-Browser\Application\brave.exe`, `%LOCALAPPDATA%\BraveSoftware\Brave-Browser\Application\brave.exe`},
	"yandex":     {`%LOCALAPPDATA%\Yandex\YandexBrowser\Application\browser.exe`, `%ProgramFiles%\Yandex\YandexBrowser\Application\browser.exe`, `%ProgramFiles(x86)%\Yandex\YandexBrowser\Application\browser.exe`},
}

func appExe(id string) string {
	if id == "raven" {
		return findExe(ravenDir(), "raven")
	}
	for _, p := range appPaths[id] {
		full := os.ExpandEnv(strings.NewReplacer("%ProgramFiles(x86)%", "${ProgramFiles(x86)}", "%ProgramFiles%", "${ProgramFiles}", "%LOCALAPPDATA%", "${LOCALAPPDATA}").Replace(p))
		if _, err := os.Stat(full); err == nil {
			return full
		}
	}
	return ""
}
func appInstalled(id string) bool { return appExe(id) != "" }

func apiAppsList() []map[string]interface{} {
	dir := installersDir()
	res := []map[string]interface{}{}
	for _, a := range appsCatalog {
		c := map[string]interface{}{}
		for k, v := range a {
			c[k] = v
		}
		c["offline"] = hasInstaller(dir, a["id"].(string)) || hasInstaller(downloadDir(), a["id"].(string))
		exe := appExe(a["id"].(string))
		c["installed"] = exe != ""
		c["path"] = exe
		delete(c, "winget")
		delete(c, "url")
		res = append(res, c)
	}
	return res
}
func hasInstaller(dir, id string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		n := strings.ToLower(e.Name())
		if strings.Contains(n, id) && (strings.HasSuffix(n, ".exe") || strings.HasSuffix(n, ".msi") || strings.HasSuffix(n, ".zip")) {
			return true
		}
	}
	return false
}
func findApp(id string) map[string]interface{} {
	for _, a := range appsCatalog {
		if a["id"] == id {
			return a
		}
	}
	return nil
}

/* ============ Защита (только статус) ============ */
func apiDefenderStatus() map[string]interface{} {
	out := psLine("try{$s=Get-MpComputerStatus; \"$($s.RealTimeProtectionEnabled)|$($s.IsTamperProtected)\"}catch{'?|?'}")
	parts := strings.Split(out, "|")
	rt, tp := "", ""
	if len(parts) == 2 {
		rt, tp = parts[0], parts[1]
	}
	return map[string]interface{}{
		"realtime": strings.EqualFold(strings.TrimSpace(rt), "True"),
		"tamper":   strings.EqualFold(strings.TrimSpace(tp), "True"),
	}
}
func apiOpenDefender() bool {
	run("cmd", "/c", "start", "windowsdefender:")
	logLine("INFO", "Открыто окно «Безопасность Windows»")
	return true
}

/* ============ Драйверы ============ */
// Ключевые устройства: видеокарта, сеть, звук, Bluetooth, чипсет — с версией и датой драйвера.
// Встроенные драйверы Microsoft не показываем: они обновляются вместе с Windows (и у них фиктивные даты 2006 года).
func apiDriverList() []map[string]interface{} {
	out := psLine(`
Get-CimInstance Win32_PnPSignedDriver | Where-Object {
  $_.DeviceName -and $_.DriverVersion -and $_.DriverProviderName -notmatch '^Microsoft' -and $_.DeviceID -match '^(PCI|USB|HDAUDIO|BTH|SWD\\DRIVERENUM)' -and
  ($_.DeviceClass -in 'DISPLAY','NET','MEDIA','BLUETOOTH' -or ($_.DeviceClass -eq 'SYSTEM' -and $_.DeviceName -match 'chipset|SMBus|PSP|GPIO|I2C|Management Engine|Serial IO|Platform'))
} | Sort-Object DeviceName -Unique | Select-Object -First 30 DeviceName, DriverVersion, DriverProviderName, DeviceClass,
  @{n='Date';e={ if($_.DriverDate){ $_.DriverDate.ToString('yyyy-MM-dd') } else { '' } }} | ConvertTo-Json -Compress`)
	if strings.HasPrefix(out, "{") {
		out = "[" + out + "]"
	}
	var devs []struct{ DeviceName, DriverVersion, DriverProviderName, DeviceClass, Date string }
	json.Unmarshal([]byte(out), &devs)
	res := []map[string]interface{}{}
	for _, d := range devs {
		age := -1
		if t, err := time.Parse("2006-01-02", d.Date); err == nil {
			age = int(time.Since(t).Hours() / 24 / 30)
		}
		res = append(res, map[string]interface{}{
			"name": d.DeviceName, "cur": d.DriverVersion, "provider": d.DriverProviderName,
			"cls": strings.ToLower(d.DeviceClass), "date": d.Date, "age": age,
		})
	}
	return res
}

// Поиск обновлений драйверов: Windows Update + драйвер NVIDIA с серверов NVIDIA (только поиск). Занимает 10–60 секунд.
func apiDriverScan() map[string]interface{} {
	type nvRes struct {
		d   *nvDriver
		err error
	}
	nvCh := make(chan nvRes, 1)
	go func() { d, err := nvCheck(); nvCh <- nvRes{d, err} }()

	out, _ := ps(`
$ErrorActionPreference='Stop'
try{
  $s=New-Object -ComObject Microsoft.Update.Session
  $r=$s.CreateUpdateSearcher().Search("IsInstalled=0 and Type='Driver'")
  $list=@(foreach($u in $r.Updates){
    [pscustomobject]@{id=$u.Identity.UpdateID; title=$u.Title; provider=$u.DriverProvider; model=$u.DriverModel; cls=$u.DriverClass;
      date=$(if($u.DriverVerDate){ ([datetime]$u.DriverVerDate).ToString('yyyy-MM-dd') } else { '' })}
  })
  ConvertTo-Json -Compress -InputObject @{ok=$true; list=$list}
}catch{ ConvertTo-Json -Compress -InputObject @{ok=$false; msg=$_.Exception.Message} }`)
	var r struct {
		Ok   bool                     `json:"ok"`
		Msg  string                   `json:"msg"`
		List []map[string]interface{} `json:"list"`
	}
	wuOk := json.Unmarshal([]byte(strings.TrimSpace(out)), &r) == nil && r.Ok
	if !wuOk {
		logLine("ERR", "Поиск драйверов в Windows Update не удался: "+r.Msg)
	} else {
		logLine("INFO", fmt.Sprintf("Windows Update: найдено обновлений драйверов — %d", len(r.List)))
	}
	list := r.List
	if list == nil {
		list = []map[string]interface{}{}
	}

	res := map[string]interface{}{"ok": wuOk, "list": list}
	if nv := <-nvCh; nv.err != nil {
		logLine("WARN", "NVIDIA: не удалось проверить драйвер — "+nv.err.Error())
	} else if nv.d != nil {
		upd := nvNewer(nv.d.Latest, nv.d.Installed)
		res["nvidia"] = map[string]interface{}{"gpu": nv.d.GPU, "installed": nv.d.Installed, "latest": nv.d.Latest, "update": upd}
		logLine("INFO", "NVIDIA: установлен "+orDash(nv.d.Installed)+", последний "+nv.d.Latest)
		if upd {
			res["list"] = append([]map[string]interface{}{{
				"id": "nvidia:" + nv.d.Latest, "title": "NVIDIA Game Ready " + nv.d.Latest + " — " + nv.d.GPU,
				"provider": "NVIDIA · " + nv.d.Size + " · с серверов NVIDIA", "date": nv.d.Date,
			}}, list...)
		}
		res["ok"] = true
	}
	if res["ok"] == false {
		res["msg"] = "Windows Update не ответил. Проверьте интернет и что служба «Центр обновления Windows» не отключена."
	}
	return res
}

// Установка выбранных обновлений драйверов (по UpdateID из apiDriverScan)
func apiUpdateDrivers(ids []string) map[string]interface{} {
	if len(ids) == 0 {
		return map[string]interface{}{"ok": false, "msg": "Не выбрано ни одного драйвера"}
	}
	msgs, okAny, reboot := []string{}, false, false
	wuIDs := []string{}
	for _, id := range ids {
		if !strings.HasPrefix(id, "nvidia:") {
			wuIDs = append(wuIDs, id)
			continue
		}
		d, err := nvCheck()
		if err == nil && d != nil {
			err = nvInstall(d)
		}
		if err != nil {
			logLine("ERR", "NVIDIA: "+err.Error())
			msgs = append(msgs, "Драйвер NVIDIA не установлен: "+err.Error()+".")
		} else {
			okAny, reboot = true, true
			msgs = append(msgs, "Драйвер NVIDIA "+d.Latest+" установлен.")
		}
	}
	if len(wuIDs) > 0 {
		r := wuInstallDrivers(wuIDs)
		msgs = append(msgs, r["msg"].(string))
		okAny = okAny || r["ok"].(bool)
		reboot = reboot || r["reboot"].(bool)
	}
	return map[string]interface{}{"ok": okAny, "msg": strings.Join(msgs, " "), "reboot": reboot}
}

func wuInstallDrivers(ids []string) map[string]interface{} {
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		quoted = append(quoted, "'"+strings.ReplaceAll(id, "'", "")+"'")
	}
	logLine("INFO", fmt.Sprintf("=== Установка драйверов из Windows Update (%d) ===", len(ids)))
	out, _ := ps(fmt.Sprintf(`
$ErrorActionPreference='Stop'
try{
  $want=@(%s)
  $s=New-Object -ComObject Microsoft.Update.Session
  $r=$s.CreateUpdateSearcher().Search("IsInstalled=0 and Type='Driver'")
  $c=New-Object -ComObject Microsoft.Update.UpdateColl
  foreach($u in $r.Updates){ if($want -contains $u.Identity.UpdateID){ if(-not $u.EulaAccepted){$u.AcceptEula()}; $c.Add($u) | Out-Null } }
  if($c.Count -eq 0){ '{"ok":false,"done":0,"total":0,"reboot":false}'; return }
  $d=$s.CreateUpdateDownloader(); $d.Updates=$c; $d.Download() | Out-Null
  $i=$s.CreateUpdateInstaller(); $i.Updates=$c; $res=$i.Install()
  $ok=0; for($k=0;$k -lt $c.Count;$k++){ if($res.GetUpdateResult($k).ResultCode -eq 2){$ok++} }
  ConvertTo-Json -Compress -InputObject @{ok=($ok -gt 0); done=$ok; total=$c.Count; reboot=$res.RebootRequired}
}catch{ ConvertTo-Json -Compress -InputObject @{ok=$false; done=0; total=0; reboot=$false; msg=$_.Exception.Message} }`, strings.Join(quoted, ",")))
	var r struct {
		Ok          bool
		Done, Total int
		Reboot      bool
		Msg         string
	}
	json.Unmarshal([]byte(strings.TrimSpace(out)), &r)
	if r.Total == 0 {
		logLine("ERR", "Драйверы не установлены: "+orDash(r.Msg))
		return map[string]interface{}{"ok": false, "msg": "Драйверы из Windows Update не установлены. Подробности — в Журнале.", "reboot": false}
	}
	logLine("OK", fmt.Sprintf("Установлено драйверов: %d из %d", r.Done, r.Total))
	msg := fmt.Sprintf("Из Windows Update установлено драйверов: %d из %d.", r.Done, r.Total)
	return map[string]interface{}{"ok": r.Ok, "msg": msg, "reboot": r.Reboot}
}
func apiBackupDrivers() bool {
	safe("Резервная копия драйверов", func() {
		dst := filepath.Join(kernDir, "DriverBackup")
		os.MkdirAll(dst, 0755)
		run("dism.exe", "/online", "/export-driver", "/destination:"+dst)
		logLine("OK", "Драйверы сохранены в "+dst)
	})
	return true
}

/* ============ Авто-очистка ============ */
func apiToggleAutoClean() bool {
	exe, _ := os.Executable()
	task := psLine(`(Get-ScheduledTask -TaskName 'KERNEL-AutoClean' -ErrorAction SilentlyContinue).TaskName`)
	if task != "" {
		run("schtasks", "/delete", "/tn", "KERNEL-AutoClean", "/f")
		logLine("INFO", "Авто-очистка выключена")
	} else {
		run("schtasks", "/create", "/tn", "KERNEL-AutoClean", "/tr", fmt.Sprintf(`"%s" --autoclean`, exe),
			"/sc", "weekly", "/d", "SUN", "/st", "12:00", "/rl", "highest", "/f")
		logLine("OK", "Авто-очистка включена (раз в неделю)")
	}
	return true
}

/* ============ Журнал API ============ */
func apiLogLines() []LogLine {
	logMu.Lock()
	defer logMu.Unlock()
	out := make([]LogLine, len(logBuf))
	copy(out, logBuf)
	return out
}

/* ============ Фатальная ошибка ============ */
func showFatal(msg string) {
	logLine("ERR", msg)
	run("powershell.exe", "-NoProfile", "-Command",
		fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework;[System.Windows.MessageBox]::Show('%s','KERNEL')`, msg))
}
