package i18n

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

var lang = "en"

func Init(override string) {
	switch strings.ToLower(override) {
	case "ru", "en":
		lang = strings.ToLower(override)
		return
	}
	if systemRussian() {
		lang = "ru"
	}
}

func Lang() string { return lang }

func systemRussian() bool {
	k := windows.NewLazySystemDLL("kernel32.dll")
	p := k.NewProc("GetUserDefaultUILanguage")
	if p.Find() != nil {
		return false
	}
	id, _, _ := p.Call()
	switch id & 0x3FF {
	case 0x19, 0x22, 0x23, 0x3F, 0x43:
		return true
	}
	return false
}

func T(key string, args ...any) string {
	m := en
	if lang == "ru" {
		m = ru
	}
	s, ok := m[key]
	if !ok {
		s = en[key]
	}
	if s == "" {
		s = key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

var ru = map[string]string{
	"status.starting":  "Запуск",
	"status.searching": "Подбор стратегии… %s",
	"status.running":   "Работает",
	"status.partial":   "Работает частично",
	"status.failed":    "Нет обхода",
	"status.offline":   "Нет сети",
	"status.disabled":  "Выключено",

	"net.none": "нет сети",
	"net.lan":  "LAN %s",

	"msg.internet_down": "интернет недоступен",
	"msg.not_found":     "рабочая стратегия не найдена",
	"msg.not_working":   "не работает: %s",

	"works":     "работает",
	"not_works": "не работает",

	"ev.research":     "Переподбор по запросу",
	"ev.enabled":      "Обход включён",
	"ev.disabled":     "Обход выключен",
	"ev.net_changed":  "Сеть сменилась: %s",
	"ev.net_missing":  "Сеть не найдена",
	"ev.cached_check": "Сеть %s: есть сохранённая стратегия, проверяю",
	"ev.internet":     "Интернет недоступен",
	"ev.cached_ok":    "Сохранённая стратегия работает",
	"ev.cached_fail":  "Сохранённая стратегия не прошла проверку: %s",
	"ev.start_fail":   "Не удалось запустить winws: %s",
	"ev.net_new":      "Сеть %s: новая, начинаю подбор",
	"ev.cand_fail":    "Кандидат %d: winws не запустился: %s",
	"ev.start_abort":  "winws не запускается, подбор остановлен. Проверьте папку zapret и драйвер WinDivert",
	"ev.control_down": "Контрольные сайты недоступны, подбор остановлен",
	"ev.cand":         "Кандидат %d: %s",
	"ev.final_fail":   "Не удалось запустить итоговую стратегию: %s",
	"ev.done":         "Подбор завершён за %s, кандидатов: %d. %s",
	"ev.winws_down":   "winws остановился, перезапускаю",
	"ev.restart_fail": "Перезапуск не удался: %s",
	"ev.check_down":   "Проверка: интернет недоступен",
	"ev.check_fail":   "Проверка: %s (сбой %d из 2)",

	"ev.dns_blocked":      "DNS провайдера не отдаёт адрес %s, включаю защищённый DNS (%s)",
	"ev.dns_blocked_off":  "DNS провайдера не отдаёт адрес %s. Включите защищённый DNS в браузере или уберите secure_dns=off в deepmole.json",
	"ev.dns_on":           "Защищённый DNS включён: %s. Прежние настройки вернутся при выходе",
	"ev.dns_fail":         "Не удалось включить защищённый DNS: %s",
	"ev.dns_off":          "Настройки DNS восстановлены",
	"ev.dns_restored":     "Восстановлены настройки DNS, оставшиеся с прошлого запуска",
	"ev.dns_restore_fail": "Не удалось восстановить настройки DNS: %s",
	"ev.dns_auto":         "Защищённый DNS: включится сам, если провайдер подменяет DNS",
	"ev.dns_manual":       "Защищённый DNS выключен, настройки DNS не трогаются",
	"ui.secure_dns":       "Защищённый DNS",

	"ui.update_btn":    "Обновить до %s",
	"ui.update_check":  "Проверить обновления",
	"ui.update_ask":    "Доступна версия %s, у вас %s.\n\nУстановить сейчас? DeepMole перезапустится.",
	"ui.update_none":   "Обновлений нет, у вас последняя версия %s.",
	"ui.update_err":    "Не удалось проверить обновления:\n%s",
	"ev.update_found":  "Доступна версия %s: %s",
	"ev.update_dl":     "Загрузка обновления %s: %d%%",
	"ev.update_apply":  "Устанавливаю обновление, программа перезапустится",
	"ev.update_fail":   "Обновление не удалось: %s",
	"ev.update_latest": "Обновлений нет, у вас последняя версия",

	"dns.system":  "системный",
	"dns.secure":  "защищённый, %s",
	"dns.blocked": "подменяется провайдером",

	"main.no_zapret": "Папка zapret не найдена. Положите bin/ и lists/ рядом с deepmole.exe или укажите zapret_dir в deepmole.json.",
	"main.admin":     "Нужны права администратора: запустите консоль от имени администратора.",
	"main.running":   "DeepMole уже запущен. Откройте его через значок в трее.",

	"ui.network":        "Сеть: %s",
	"ui.research":       "Переподобрать",
	"ui.disable":        "Выключить",
	"ui.enable":         "Включить",
	"ui.log":            "Лог",
	"ui.autostart":      "Автозапуск",
	"ui.open":           "Открыть окно",
	"ui.disable_bypass": "Выключить обход",
	"ui.enable_bypass":  "Включить обход",
	"ui.open_log":       "Открыть лог",
	"ui.quit":           "Выход",
	"ui.autostart_err":  "Не удалось изменить автозапуск:\n%s",
	"ui.research_tip":   "Заново подобрать стратегию для этой сети",
}

var en = map[string]string{
	"status.starting":  "Starting",
	"status.searching": "Searching for a strategy… %s",
	"status.running":   "Running",
	"status.partial":   "Partially working",
	"status.failed":    "No bypass",
	"status.offline":   "No network",
	"status.disabled":  "Disabled",

	"net.none": "no network",
	"net.lan":  "LAN %s",

	"msg.internet_down": "internet unavailable",
	"msg.not_found":     "no working strategy found",
	"msg.not_working":   "not working: %s",

	"works":     "working",
	"not_works": "not working",

	"ev.research":     "Re-search requested",
	"ev.enabled":      "Bypass enabled",
	"ev.disabled":     "Bypass disabled",
	"ev.net_changed":  "Network changed: %s",
	"ev.net_missing":  "No network found",
	"ev.cached_check": "Network %s: saved strategy found, verifying",
	"ev.internet":     "Internet unavailable",
	"ev.cached_ok":    "Saved strategy works",
	"ev.cached_fail":  "Saved strategy failed verification: %s",
	"ev.start_fail":   "Failed to start winws: %s",
	"ev.net_new":      "Network %s: new, starting search",
	"ev.cand_fail":    "Candidate %d: winws failed to start: %s",
	"ev.start_abort":  "winws cannot start, search stopped. Check the zapret folder and the WinDivert driver",
	"ev.control_down": "Control sites unreachable, search stopped",
	"ev.cand":         "Candidate %d: %s",
	"ev.final_fail":   "Failed to start the final strategy: %s",
	"ev.done":         "Search finished in %s, candidates: %d. %s",
	"ev.winws_down":   "winws stopped, restarting",
	"ev.restart_fail": "Restart failed: %s",
	"ev.check_down":   "Check: internet unavailable",
	"ev.check_fail":   "Check: %s (failure %d of 2)",

	"ev.dns_blocked":      "The ISP's DNS gives no address for %s, enabling secure DNS (%s)",
	"ev.dns_blocked_off":  "The ISP's DNS gives no address for %s. Enable secure DNS in the browser or remove secure_dns=off from deepmole.json",
	"ev.dns_on":           "Secure DNS enabled: %s. Previous settings are restored on exit",
	"ev.dns_fail":         "Failed to enable secure DNS: %s",
	"ev.dns_off":          "DNS settings restored",
	"ev.dns_restored":     "Restored DNS settings left over from the previous run",
	"ev.dns_restore_fail": "Failed to restore DNS settings: %s",
	"ev.dns_auto":         "Secure DNS: turns on by itself when the ISP hijacks DNS",
	"ev.dns_manual":       "Secure DNS disabled, DNS settings are left untouched",
	"ui.secure_dns":       "Secure DNS",

	"ui.update_btn":    "Update to %s",
	"ui.update_check":  "Check for updates",
	"ui.update_ask":    "Version %s is available, you have %s.\n\nInstall now? DeepMole will restart.",
	"ui.update_none":   "No updates, you have the latest version %s.",
	"ui.update_err":    "Failed to check for updates:\n%s",
	"ev.update_found":  "Version %s is available: %s",
	"ev.update_dl":     "Downloading update %s: %d%%",
	"ev.update_apply":  "Installing the update, the program will restart",
	"ev.update_fail":   "Update failed: %s",
	"ev.update_latest": "No updates, you have the latest version",

	"dns.system":  "system",
	"dns.secure":  "secure, %s",
	"dns.blocked": "hijacked by the ISP",

	"main.no_zapret": "zapret folder not found. Put bin/ and lists/ next to deepmole.exe or set zapret_dir in deepmole.json.",
	"main.admin":     "Administrator rights are required: run the console as administrator.",
	"main.running":   "DeepMole is already running. Open it from the tray icon.",

	"ui.network":        "Network: %s",
	"ui.research":       "Re-search",
	"ui.disable":        "Disable",
	"ui.enable":         "Enable",
	"ui.log":            "Log",
	"ui.autostart":      "Autostart",
	"ui.open":           "Open window",
	"ui.disable_bypass": "Disable bypass",
	"ui.enable_bypass":  "Enable bypass",
	"ui.open_log":       "Open log",
	"ui.quit":           "Exit",
	"ui.autostart_err":  "Failed to change autostart:\n%s",
	"ui.research_tip":   "Search for a strategy on this network again",
}
