package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

/* ===== Исходные значения настроек: откат возвращает то, что было ДО KERNEL ===== */
// Хранятся в %LOCALAPPDATA%\KERNEL\tweaks-backup.json. Ключ — произвольное имя,
// значение — исходное значение строкой или "-" (значения не было).

var origMu sync.Mutex

func origPath() string { return filepath.Join(kernDir, "tweaks-backup.json") }

func origLoad() map[string]string {
	m := map[string]string{}
	if data, err := os.ReadFile(origPath()); err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}

// Запоминаем исходное значение один раз (повторное применение твика его не перезаписывает)
func origSave(key, val string) {
	origMu.Lock()
	defer origMu.Unlock()
	m := origLoad()
	if _, ok := m[key]; ok {
		return
	}
	m[key] = val
	if data, err := json.Marshal(m); err == nil {
		os.WriteFile(origPath(), data, 0644)
	}
}

// Достаём исходное значение и забываем его
func origTake(key string) (string, bool) {
	origMu.Lock()
	defer origMu.Unlock()
	m := origLoad()
	v, ok := m[key]
	if ok {
		delete(m, key)
		if data, err := json.Marshal(m); err == nil {
			os.WriteFile(origPath(), data, 0644)
		}
	}
	return v, ok
}

// DWORD: запомнить исходное значение и записать новое
func regDwordBackup(root, path, name string, val int) {
	cur := "-"
	if v, ok := regReadDword(root, path, name); ok {
		cur = strconv.Itoa(int(v))
	}
	origSave(root+`\`+path+`\`+name, cur)
	regDword(root, path, name, val)
}

// DWORD: вернуть исходное значение. Если исходного нет (твик ставила старая версия
// или его не применяли) — ничего не трогаем, чтобы не сломать чужую настройку.
func regDwordRestore(root, path, name string) {
	v, ok := origTake(root + `\` + path + `\` + name)
	if !ok {
		return
	}
	if v == "-" {
		regDel(root, path, name)
		return
	}
	if n, err := strconv.Atoi(v); err == nil {
		regDword(root, path, name, n)
	}
}

// Название процессора из реестра (быстро, без PowerShell)
func cpuName() string {
	s, _ := regReadString("HKLM", `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "ProcessorNameString")
	return strings.ToLower(strings.TrimSpace(s))
}

// Ryzen X3D с двумя блоками ядер (7900X3D, 7950X3D, 9900X3D, 9950X3D): драйвер AMD «паркует»
// блок без 3D-кэша в играх только в сбалансированной схеме питания. «Максимальная
// производительность» отключает парковку — игра расползается на оба блока и FPS падает.
func isDualCcdX3D() bool {
	n := cpuName()
	for _, m := range []string{"7900x3d", "7950x3d", "9900x3d", "9950x3d"} {
		if strings.Contains(n, m) {
			return true
		}
	}
	return false
}
