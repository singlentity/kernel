package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

/* ===== Автообновление драйвера видеокарты NVIDIA (напрямую с серверов NVIDIA) =====
   Windows Update отдаёт драйверы NVIDIA с опозданием на месяцы, поэтому версию берём
   из того же сервиса, что и сайт nvidia.com: по названию видеокарты находим её ID,
   затем последнюю Game Ready (WHQL, DCH) версию и прямую ссылку на установщик. */

type nvDriver struct {
	GPU       string // название видеокарты
	Installed string // установленная версия в формате NVIDIA (621.01)
	Latest    string
	URL       string
	Size      string
	Date      string
}

var nvHTTP = &http.Client{Timeout: 30 * time.Second}

// Версия драйвера Windows (32.0.16.2101) → версия NVIDIA (621.01): последние 5 цифр
func nvVersion(winVer string) string {
	d := strings.ReplaceAll(winVer, ".", "")
	if len(d) < 5 {
		return ""
	}
	d = d[len(d)-5:]
	return d[:3] + "." + d[3:]
}

// Установленная видеокарта NVIDIA и её драйвер
func nvInstalled() (name, ver string) {
	out := psLine(`Get-CimInstance Win32_VideoController | Where-Object { $_.Name -match 'NVIDIA|GeForce|Quadro|RTX' } | Select-Object -First 1 | ForEach-Object { $_.Name + '|' + $_.DriverVersion }`)
	if p := strings.SplitN(out, "|", 2); len(p) == 2 {
		return strings.TrimSpace(p[0]), nvVersion(strings.TrimSpace(p[1]))
	}
	return "", ""
}

func nvNorm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimSpace(strings.TrimPrefix(s, "nvidia "))
}

// ID видеокарты в базе NVIDIA по её названию
func nvProductID(gpu string) (string, error) {
	resp, err := nvHTTP.Get("https://www.nvidia.com/Download/API/lookupValueSearch.aspx?TypeID=3")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var x struct {
		Values []struct {
			Name  string `xml:"Name"`
			Value string `xml:"Value"`
		} `xml:"LookupValues>LookupValue"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&x); err != nil {
		return "", err
	}
	want := nvNorm(gpu)
	for _, v := range x.Values {
		if nvNorm(v.Name) == want {
			return v.Value, nil
		}
	}
	return "", fmt.Errorf("видеокарта «%s» не найдена в базе NVIDIA", gpu)
}

// Последний драйвер для видеокарты: nil, если NVIDIA нет; ошибка — если сервис недоступен
func nvCheck() (*nvDriver, error) {
	gpu, inst := nvInstalled()
	if gpu == "" {
		return nil, nil
	}
	pfid, err := nvProductID(gpu)
	if err != nil {
		return nil, err
	}
	q := "https://gfwsl.geforce.com/services_toolkit/services/com/nvidia/services/AjaxDriverService.php?func=DriverManualLookup" +
		"&pfid=" + pfid + "&osID=57&languageCode=1033&isWHQL=1&dch=1&sort1=0&numberOfResults=1"
	resp, err := nvHTTP.Get(q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r struct {
		IDS []struct {
			DownloadInfo struct {
				Version, DownloadURL, DownloadURLFileSize, ReleaseDateTime string
			} `json:"downloadInfo"`
		} `json:"IDS"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || len(r.IDS) == 0 {
		return nil, fmt.Errorf("сервис NVIDIA не вернул драйвер")
	}
	di := r.IDS[0].DownloadInfo
	date := di.ReleaseDateTime
	if t, err := time.Parse("Mon Jan 2, 2006", di.ReleaseDateTime); err == nil {
		date = t.Format("2006-01-02")
	}
	return &nvDriver{GPU: gpu, Installed: inst, Latest: di.Version, URL: di.DownloadURL, Size: di.DownloadURLFileSize, Date: date}, nil
}

// Сравнение версий вида 621.01
func nvNewer(latest, inst string) bool {
	var a1, a2, b1, b2 int
	fmt.Sscanf(latest, "%d.%d", &a1, &a2)
	fmt.Sscanf(inst, "%d.%d", &b1, &b2)
	return a1 > b1 || (a1 == b1 && a2 > b2)
}

var nvURLRe = regexp.MustCompile(`^https://[a-z0-9.-]+\.nvidia\.com/`)

// Скачать, проверить цифровую подпись NVIDIA и установить без окон
func nvInstall(d *nvDriver) error {
	if u, err := url.Parse(d.URL); err != nil || !nvURLRe.MatchString(d.URL) || u.Scheme != "https" {
		return fmt.Errorf("подозрительная ссылка на драйвер: %s", d.URL)
	}
	dir := filepath.Join(os.Getenv("TEMP"), "KERNEL")
	os.MkdirAll(dir, 0755)
	exe := filepath.Join(dir, "nvidia-"+d.Latest+".exe")

	logLine("INFO", "NVIDIA: скачиваю драйвер "+d.Latest+" ("+d.Size+")")
	dl := &http.Client{Timeout: 60 * time.Minute}
	resp, err := dl.Get(d.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("сервер NVIDIA ответил %d", resp.StatusCode)
	}
	f, err := os.Create(exe)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(exe)
		return err
	}
	defer os.Remove(exe)

	// Запускаем только файл с действительной подписью NVIDIA
	sig := psLine(fmt.Sprintf(`$s=Get-AuthenticodeSignature -LiteralPath '%s'; "$($s.Status)|$($s.SignerCertificate.Subject)"`, strings.ReplaceAll(exe, "'", "''")))
	if !strings.HasPrefix(sig, "Valid|") || !strings.Contains(sig, "NVIDIA Corporation") {
		return fmt.Errorf("подпись установщика не прошла проверку (%s) — установка отменена", sig)
	}

	logLine("INFO", "NVIDIA: устанавливаю драйвер "+d.Latest+" (тихий режим, настройки сохраняются)")
	// -s: без окон, -noreboot: перезагрузку предложит KERNEL, -noeula: без окна лицензии
	_, runErr := run(exe, "-s", "-noreboot", "-noeula")
	if _, now := nvInstalled(); now != d.Latest {
		if runErr != nil {
			return fmt.Errorf("установщик NVIDIA завершился с ошибкой (%v), стоит версия %s", runErr, orDash(now))
		}
		// установщик отработал без ошибок, но система ещё показывает старую версию — она сменится после перезагрузки
		logLine("WARN", "NVIDIA: новая версия появится после перезагрузки")
	}
	logLine("OK", "NVIDIA: установлен драйвер "+d.Latest)
	return nil
}
