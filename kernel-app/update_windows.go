package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

/* ===== Проверка обновлений ===== */

// Текущая версия этой сборки. Поднимай при каждом релизе.
const AppVersion = "1.6.0"

// Адрес манифеста с информацией о последней версии (простой JSON).
// ЗАМЕНИ на свой: GitHub raw, Gist raw или любой публичный URL, отдающий JSON вида:
//   {"version":"1.6.0","url":"https://t.me/kernel1lab/123","notes":"Что нового"}
// url — ссылка, которую откроет кнопка «Скачать» (можно на пост в Telegram с .exe).
const UpdateManifestURL = "https://raw.githubusercontent.com/singlentity/kernel/main/version.json"

type updateManifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	Notes   string `json:"notes"`
}

func apiAppVersion() string { return AppVersion }

// Возвращает инфо об обновлении. При любой ошибке — available:false (пользователя не беспокоим).
func apiCheckUpdate() map[string]interface{} {
	none := map[string]interface{}{"available": false, "current": AppVersion}
	req, err := http.NewRequest("GET", UpdateManifestURL, nil)
	if err != nil {
		return none
	}
	req.Header.Set("User-Agent", "KERNEL/"+AppVersion)
	resp, err := httpc.Do(req)
	if err != nil {
		return none
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return none
	}
	var m updateManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil || m.Version == "" {
		return none
	}
	if compareVersions(m.Version, AppVersion) <= 0 {
		return none // не новее
	}
	url := m.URL
	if !strings.HasPrefix(url, "https://") {
		url = "https://t.me/kernel1lab"
	}
	return map[string]interface{}{
		"available": true, "current": AppVersion,
		"latest": m.Version, "url": url, "notes": m.Notes,
	}
}

// Сравнение версий "a.b.c": >0 если a новее b, 0 если равны, <0 если старше
func compareVersions(a, b string) int {
	pa, pb := splitVer(a), splitVer(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		va, vb := 0, 0
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if va != vb {
			if va > vb {
				return 1
			}
			return -1
		}
	}
	return 0
}
func splitVer(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(strings.TrimFunc(p, func(r rune) bool { return r < '0' || r > '9' }))
		out = append(out, n)
	}
	return out
}
