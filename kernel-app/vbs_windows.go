package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

/* ===== Твик «Отключить VBS (виртуализация безопасности)» ===== */

const vbsDGPath = `SYSTEM\CurrentControlSet\Control\DeviceGuard`
const vbsHVCIPath = `SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity`

// Сохранённое исходное состояние (чтобы откат возвращал именно его, а не жёстко 1)
type vbsBackup struct {
	Saved bool   `json:"saved"`
	HLT   string `json:"hlt"`  // исходный hypervisorlaunchtype (off/auto/on), "" если не прочитан
	Vbs   int    `json:"vbs"`  // исходный EnableVirtualizationBasedSecurity, -1 если ключа не было
	Hvci  int    `json:"hvci"` // исходный HypervisorEnforcedCodeIntegrity\Enabled, -1 если ключа не было
}

func vbsBackupPath() string { return filepath.Join(kernDir, "vbs-backup.json") }

// Текущий hypervisorlaunchtype из BCD (best-effort; на локализованной Windows поле может не распознаться)
func readBcdHLT() string {
	out, _ := run("bcdedit.exe", "/enum", "{current}")
	for _, ln := range strings.Split(out, "\n") {
		l := strings.ToLower(strings.TrimSpace(ln))
		if strings.Contains(l, "hypervisorlaunchtype") {
			f := strings.Fields(l)
			if len(f) > 0 {
				v := f[len(f)-1]
				if v == "off" || v == "auto" || v == "on" {
					return v
				}
			}
		}
	}
	return ""
}

// Сохраняем исходные значения один раз (при первом применении)
func vbsSaveBackupIfNeeded() {
	if _, err := os.Stat(vbsBackupPath()); err == nil {
		return // бэкап уже есть — не перезаписываем
	}
	b := vbsBackup{Saved: true, HLT: readBcdHLT(), Vbs: -1, Hvci: -1}
	if v, ok := regReadDword("HKLM", vbsDGPath, "EnableVirtualizationBasedSecurity"); ok {
		b.Vbs = int(v)
	}
	if v, ok := regReadDword("HKLM", vbsHVCIPath, "Enabled"); ok {
		b.Hvci = int(v)
	}
	if data, err := json.Marshal(b); err == nil {
		os.WriteFile(vbsBackupPath(), data, 0644)
	}
	logLine("INFO", "VBS: исходное состояние сохранено (hlt="+orDash(b.HLT)+")")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// Применение: отключаем VBS
func vbsApply() {
	vbsSaveBackupIfNeeded()
	run("bcdedit.exe", "/set", "hypervisorlaunchtype", "off")
	regDword("HKLM", vbsDGPath, "EnableVirtualizationBasedSecurity", 0)
	regDword("HKLM", vbsHVCIPath, "Enabled", 0)
}

// Откат: возвращаем именно исходное состояние (или безопасные значения, если бэкапа нет)
func vbsRevert() {
	hlt := "auto"
	var b vbsBackup
	if data, err := os.ReadFile(vbsBackupPath()); err == nil && json.Unmarshal(data, &b) == nil && b.Saved {
		if b.HLT != "" {
			hlt = b.HLT
		}
		if b.Vbs < 0 {
			regDel("HKLM", vbsDGPath, "EnableVirtualizationBasedSecurity")
		} else {
			regDword("HKLM", vbsDGPath, "EnableVirtualizationBasedSecurity", b.Vbs)
		}
		if b.Hvci < 0 {
			regDel("HKLM", vbsHVCIPath, "Enabled")
		} else {
			regDword("HKLM", vbsHVCIPath, "Enabled", b.Hvci)
		}
	} else {
		// бэкапа нет — возвращаем значения по умолчанию
		regDword("HKLM", vbsDGPath, "EnableVirtualizationBasedSecurity", 1)
		regDword("HKLM", vbsHVCIPath, "Enabled", 1)
	}
	run("bcdedit.exe", "/set", "hypervisorlaunchtype", hlt)
	os.Remove(vbsBackupPath())
}

// Применён ли твик (наши настройки записаны: VBS отключён в политике)
func vbsTweakApplied() bool {
	v, ok := regReadDword("HKLM", vbsDGPath, "EnableVirtualizationBasedSecurity")
	return ok && v == 0
}

// Текущее состояние VBS через WMI Win32_DeviceGuard
func apiVbsStatus() map[string]interface{} {
	out := psLine(`try{(Get-CimInstance -Namespace root\Microsoft\Windows\DeviceGuard -ClassName Win32_DeviceGuard -ErrorAction Stop).VirtualizationBasedSecurityStatus}catch{'-'}`)
	st := -1
	switch strings.TrimSpace(out) {
	case "0":
		st = 0
	case "1":
		st = 1
	case "2":
		st = 2
	}
	text := "не удалось определить"
	switch st {
	case 0:
		text = "выключен"
	case 1:
		text = "включён, но не запущен"
	case 2:
		text = "работает"
	}
	return map[string]interface{}{"status": st, "text": text}
}

// Отдельная кнопка «Вернуть как было»
func apiVbsRevert() Result {
	logLine("INFO", "=== VBS: возврат как было ===")
	ok := true
	safeOk := func(what string, f func()) {
		defer func() {
			if r := recover(); r != nil {
				ok = false
				logLine("ERR", what+" — "+itoaAny(r))
			}
		}()
		f()
		logLine("OK", what)
	}
	safeOk("VBS: возврат исходного состояния", vbsRevert)
	if !ok {
		return errR("Не удалось вернуть VBS. Нужны права администратора. Подробности — в Журнале.")
	}
	return okR("VBS возвращён к исходному состоянию. Изменения вступят в силу после перезагрузки.")
}

func itoaAny(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "ошибка"
}
