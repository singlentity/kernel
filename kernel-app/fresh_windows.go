package main

import (
	"os"
	"strings"
	"os/exec"
	"path/filepath"
)

/* ===== Вкладка «Чистый старт»: базовая настройка свежей Windows ===== */

type FreshItem struct {
	ID, Name, Desc string
	On             bool // рекомендация по умолчанию (не используется для состояния)
	explorer       bool // требуется перезапуск Проводника
	apply          func()
	revert         func()
	applied        func() bool // реальное состояние в системе
}

// GUID значков рабочего стола
const (
	guidRecycle = "{645FF040-5081-101B-9F08-00AA002F954E}"
	guidThisPC  = "{20D04FE0-3AEA-1069-A2D8-08002B30309D}"
	guidUser    = "{59031a47-3f72-44a7-89c5-5595fe6b30ee}"
	guidNetwork = "{F02C1A0D-BE21-4350-88B0-7367FC96EF3C}"
	guidCPanel  = "{5399E694-6CE5-4D6C-8FCE-1D8870FDCBA0}"
	guidSpot    = "{2cc5ca98-6485-489a-920e-b3e88a6ccce3}"
)

func hideDesktopIcon(guid string, hide int) {
	for _, root := range []string{
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\ClassicStartMenu`,
	} {
		reg("add", root, "/v", guid, "/t", "REG_DWORD", "/d", itoa(hide), "/f")
	}
}
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	return "1"
}

func freshDefs() []FreshItem {
	adv := `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
	cdm := `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`
	return []FreshItem{
		{ID: "startclean", Name: "Чистое меню Пуск", On: true, explorer: true,
			Desc: "Убирает из Пуска рекомендации, рекламу, «недавно добавленные» и советы Microsoft. Остаётся только то, что закрепили вы.",
			apply: func() {
				regDword("HKCU", adv, "Start_IrisRecommendations", 0)
				regDword("HKCU", adv, "Start_TrackDocs", 0)
				regDword("HKCU", adv, "Start_TrackProgs", 0)
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Windows\Explorer`, "HideRecommendedSection", 1)
				regDword("HKCU", `SOFTWARE\Policies\Microsoft\Windows\Explorer`, "HideRecommendedSection", 1)
				for _, v := range []string{"SubscribedContent-338388Enabled", "SubscribedContent-338389Enabled", "SubscribedContent-310093Enabled", "SystemPaneSuggestionsEnabled", "SilentInstalledAppsEnabled", "OemPreInstalledAppsEnabled", "PreInstalledAppsEnabled"} {
					regDword("HKCU", cdm, v, 0)
				}
				regDword("HKLM", `SOFTWARE\Microsoft\PolicyManager\current\device\Start`, "HideRecommendedSection", 1)
			},
			revert: func() {
				regDword("HKCU", adv, "Start_IrisRecommendations", 1)
				regDword("HKCU", adv, "Start_TrackDocs", 1)
				regDword("HKCU", adv, "Start_TrackProgs", 1)
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Windows\Explorer`, "HideRecommendedSection")
				regDel("HKCU", `SOFTWARE\Policies\Microsoft\Windows\Explorer`, "HideRecommendedSection")
			},
			applied: func() bool { return dwordIs("HKCU", adv, "Start_IrisRecommendations", 0) }},
		{ID: "cleandesktop", Name: "Чистый рабочий стол", On: true, explorer: true,
			Desc: "Убирает с рабочего стола все системные значки Windows (Корзина, Этот компьютер, Сеть, папка пользователя, «Сведения об изображении»). Ваши файлы и ярлыки не трогаются.",
			apply: func() {
				for _, g := range []string{guidRecycle, guidThisPC, guidUser, guidNetwork, guidCPanel, guidSpot} {
					hideDesktopIcon(g, 1)
				}
			},
			revert: func() {
				for _, g := range []string{guidRecycle, guidThisPC, guidUser, guidNetwork, guidCPanel, guidSpot} {
					hideDesktopIcon(g, 0)
				}
			},
			applied: func() bool {
				return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel`, guidThisPC, 1)
			}},
		{ID: "recyclebin", Name: "Убрать корзину с рабочего стола", On: false, explorer: true,
			Desc: "Прячет значок корзины с рабочего стола, чтобы не мозолил глаз (будет находиться в меню Пуск). Сама корзина продолжает работать.",
			apply: func() {
				hideDesktopIcon(guidRecycle, 1)
				pinRecycleToStart()
			},
			revert: func() {
				hideDesktopIcon(guidRecycle, 0)
				os.Remove(filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs\Корзина.lnk`))
			},
			applied: func() bool {
				return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel`, guidRecycle, 1)
			}},
		{ID: "fileext", Name: "Показывать расширения файлов", On: true, explorer: true,
			Desc: "Включает отображение .exe, .txt, .jpg в именах файлов — так видно, что ты реально открываешь. Первое, что делает любой опытный пользователь.",
			apply:   func() { regDword("HKCU", adv, "HideFileExt", 0) },
			revert:  func() { regDword("HKCU", adv, "HideFileExt", 1) },
			applied: func() bool { return dwordIs("HKCU", adv, "HideFileExt", 0) }},
		{ID: "classicmenu", Name: "Классическое контекстное меню", On: true, explorer: true,
			Desc: "Возвращает полное меню по правому клику из Windows 10 — без «Показать дополнительные параметры» в два шага.",
			apply: func() {
				k := `HKCU\SOFTWARE\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\InprocServer32`
				reg("add", k, "/ve", "/t", "REG_SZ", "/d", "", "/f")
			},
			revert: func() {
				run("reg.exe", "delete", `HKCU\SOFTWARE\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}`, "/f")
			},
			applied: func() bool {
				_, err := run("reg.exe", "query", `HKCU\SOFTWARE\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\InprocServer32`)
				return err == nil
			}},
		{ID: "darkmode", Name: "Тёмная тема Windows", On: true, explorer: false,
			Desc: "Переключает систему и приложения в тёмный режим.",
			apply: func() {
				p := `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`
				regDword("HKCU", p, "AppsUseLightTheme", 0)
				regDword("HKCU", p, "SystemUsesLightTheme", 0)
			},
			revert: func() {
				p := `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`
				regDword("HKCU", p, "AppsUseLightTheme", 1)
				regDword("HKCU", p, "SystemUsesLightTheme", 1)
			},
			applied: func() bool { return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "AppsUseLightTheme", 0) }},
		{ID: "hidewidgets", Name: "Убрать виджеты с панели задач", On: true, explorer: false,
			Desc: "Прячет кнопку погоды/новостей слева на панели задач.",
			apply: func() {
				regDword("HKCU", adv, "TaskbarDa", 0)
				regDword("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", 0)
			},
			revert: func() {
				regDword("HKCU", adv, "TaskbarDa", 1)
				regDel("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests")
			},
			applied: func() bool { return dwordIs("HKCU", adv, "TaskbarDa", 0) }},
		{ID: "hidesearch", Name: "Скрыть поиск с панели задач", On: true, explorer: false,
			Desc: "Полностью убирает строку поиска с панели задач (режим «Скрыть»). Поиск по-прежнему работает по кнопке Пуск или Win+S.",
			apply:   func() { regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Search`, "SearchboxTaskbarMode", 0) },
			revert:  func() { regDword("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Search`, "SearchboxTaskbarMode", 2) },
			applied: func() bool { return dwordIs("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Search`, "SearchboxTaskbarMode", 0) }},
		{ID: "hidetaskview", Name: "Убрать кнопку «Представление задач»", On: false, explorer: false,
			Desc: "Прячет кнопку переключения рабочих столов с панели задач.",
			apply:   func() { regDword("HKCU", adv, "ShowTaskViewButton", 0) },
			revert:  func() { regDword("HKCU", adv, "ShowTaskViewButton", 1) },
			applied: func() bool { return dwordIs("HKCU", adv, "ShowTaskViewButton", 0) }},
	}
}

func apiFreshList() []map[string]interface{} {
	res := []map[string]interface{}{}
	for _, f := range freshDefs() {
		on := false
		if f.applied != nil {
			on = f.applied()
		}
		res = append(res, map[string]interface{}{
			"id": f.ID, "name": f.Name, "desc": f.Desc, "on": on, "applied": on,
		})
	}
	return res
}

// Двустороннее применение: что отмечено — применяем, что НЕ отмечено — откатываем.
// Так состояние системы всегда совпадает с тумблерами.
func apiFreshApply(ids []string) Result {
	checked := map[string]bool{}
	for _, id := range ids {
		checked[id] = true
	}
	logLine("INFO", "=== Чистый старт: синхронизация ===")
	changed, needExplorer := 0, false
	for _, f := range freshDefs() {
		nowOn := f.applied != nil && f.applied()
		want := checked[f.ID]
		if want == nowOn {
			continue // уже в нужном состоянии
		}
		if want {
			safe("Включаю: "+f.Name, f.apply)
		} else if f.revert != nil {
			safe("Выключаю: "+f.Name, f.revert)
		} else {
			continue
		}
		changed++
		if f.explorer {
			needExplorer = true
		}
	}
	if needExplorer {
		restartExplorer()
	}
	if changed == 0 {
		return okR("Всё уже в выбранном состоянии — менять нечего.")
	}
	return okR("Готово. Изменено пунктов: " + itoa2(changed) + ". Состояние сохранено.")
}

func itoa2(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	b := []byte{}
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func restartExplorer() {
	run("taskkill", "/f", "/im", "explorer.exe")
	exec.Command(os.Getenv("WINDIR") + `\\explorer.exe`).Start()
}

/* ===== Обои ===== */
// Ставит обои: путь к файлу или "" = стандартные обои Windows 11
func apiSetWallpaper(path string) Result {
	if path == "" {
		// пробуем пользовательский wallpaper.* рядом с exe, иначе стандартные обои Windows 11
		exe, _ := os.Executable()
		dir := filepath.Dir(exe)
		for _, n := range []string{"wallpaper.jpg", "wallpaper.png", "wallpaper.jpeg"} {
			p := filepath.Join(dir, n)
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
		if path == "" {
			path = os.Getenv("WINDIR") + `\Web\Wallpaper\Windows\img0.jpg`
		}
	}
	if _, err := os.Stat(path); err != nil {
		return errR("Файл обоев не найден: " + path)
	}
	reg("add", `HKCU\Control Panel\Desktop`, "/v", "Wallpaper", "/t", "REG_SZ", "/d", path, "/f")
	reg("add", `HKCU\Control Panel\Desktop`, "/v", "WallpaperStyle", "/t", "REG_SZ", "/d", "10", "/f") // заполнение
	run("RUNDLL32.EXE", "user32.dll,UpdatePerUserSystemParameters", "1", "True")
	logLine("OK", "Обои установлены: "+path)
	return okR("Обои установлены.")
}

// Инфо по закреплению в Пуске (Windows 11 блокирует скриптовое закрепление)
func apiPinInfo() map[string]interface{} {
	return map[string]interface{}{
		"blocked": true,
		"msg":     "Windows 11 хранит закреплённые приложения в закрытом файле start2.bin и не даёт менять их скриптом. Надёжно закрепить Параметры/Проводник/Корзину в Пуск кнопкой нельзя — это ограничение Microsoft. Проще закрепить вручную: правый клик по приложению → «Закрепить на начальном экране».",
	}
}

// Создаёт ярлык «Корзина» в меню Пуск (в списке приложений) — открывает папку корзины.
// Это надёжный способ держать корзину в Пуске, т.к. закрепление-плитку Windows 11 скриптом менять не даёт.
func pinRecycleToStart() {
	progData := os.Getenv("ProgramData")
	if progData == "" {
		progData = `C:\ProgramData`
	}
	lnk := filepath.Join(progData, `Microsoft\Windows\Start Menu\Programs\Корзина.lnk`)
	explorer := filepath.Join(os.Getenv("WINDIR"), "explorer.exe")
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	ps("$s=(New-Object -ComObject WScript.Shell).CreateShortcut('" + esc(lnk) + "');" +
		"$s.TargetPath='" + esc(explorer) + "';" +
		"$s.Arguments='shell:RecycleBinFolder';" +
		"$s.IconLocation='" + esc(os.Getenv("WINDIR")+`\System32\imageres.dll`) + ",-54';" +
		"$s.Save()")
	logLine("OK", "Корзина добавлена в меню Пуск (список приложений)")
}
