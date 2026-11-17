package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Row is a single label/value line inside a card. An empty Label renders
// Value across the full card width (for prose or checklist items).
type Row struct {
	Label string
	Value string
}

// Card is a titled block of rows shown on the dashboard.
type Card struct {
	Title string
	// Hero is a big display value (e.g. the current time or temperature).
	Hero string
	// Sub is small text rendered beside/under the hero.
	Sub string
	// Icon is a weather icon name ("" for none).
	Icon string
	// Invert renders the card as white-on-black.
	Invert bool
	Rows   []Row
}

// envOr returns the environment variable's value, or def when unset.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// fetchGET returns the body of url, capped at 2 MiB, with a 15s timeout.
func fetchGET(rawurl string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, err
	}
	// Some providers (Yahoo) reject requests without a user agent.
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) kindle-dashboard")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// currentTime returns the current time in the dashboard's display timezone
// (DASH_TZ, an IANA name), defaulting to America/Sao_Paulo.
func currentTime() time.Time {
	name := envOr("DASH_TZ", "America/Sao_Paulo")
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

// dashboardTitle returns the header title (DASH_TITLE, empty until named).
func dashboardTitle() string {
	return envOr("DASH_TITLE", "")
}

var weekdaysPT = map[time.Weekday]string{
	time.Sunday:    "Domingo",
	time.Monday:    "Segunda-feira",
	time.Tuesday:   "Terça-feira",
	time.Wednesday: "Quarta-feira",
	time.Thursday:  "Quinta-feira",
	time.Friday:    "Sexta-feira",
	time.Saturday:  "Sábado",
}

var weekdaysShortPT = map[time.Weekday]string{
	time.Sunday: "dom", time.Monday: "seg", time.Tuesday: "ter",
	time.Wednesday: "qua", time.Thursday: "qui", time.Friday: "sex",
	time.Saturday: "sáb",
}

var monthsShortPT = map[time.Month]string{
	time.January: "jan", time.February: "fev", time.March: "mar",
	time.April: "abr", time.May: "mai", time.June: "jun",
	time.July: "jul", time.August: "ago", time.September: "set",
	time.October: "out", time.November: "nov", time.December: "dez",
}

// ptStamp formats the header timestamp in Portuguese, e.g. "seg 17 nov 22:15".
func ptStamp(t time.Time) string {
	return fmt.Sprintf("%s %02d %s %s",
		weekdaysShortPT[t.Weekday()], t.Day(), monthsShortPT[t.Month()], t.Format("15:04"))
}

// ---------- Clock ----------

func clockCard() Card {
	now := currentTime()
	return Card{
		Invert: true,
		Hero:   now.Format("15:04"),
		Sub: fmt.Sprintf("%s · %s · %s",
			weekdaysPT[now.Weekday()], now.Format("02/01/2006"),
			envOr("DASH_TZ_LABEL", "Brasília")),
	}
}

// ---------- Weather (Open-Meteo, no API key) ----------

type openMeteoCurrent struct {
	Temperature float64 `json:"temperature_2m"`
	Apparent    float64 `json:"apparent_temperature"`
	Humidity    int     `json:"relative_humidity_2m"`
	WeatherCode int     `json:"weather_code"`
	WindSpeed   float64 `json:"wind_speed_10m"`
}

type openMeteoDaily struct {
	TempMax []float64 `json:"temperature_2m_max"`
	TempMin []float64 `json:"temperature_2m_min"`
}

type openMeteoResp struct {
	Current openMeteoCurrent `json:"current"`
	Daily   openMeteoDaily   `json:"daily"`
}

// wmoText maps WMO weather codes to short Portuguese descriptions.
func wmoText(code int) string {
	switch {
	case code == 0:
		return "Céu limpo"
	case code == 1:
		return "Predom. limpo"
	case code == 2:
		return "Parc. nublado"
	case code == 3:
		return "Nublado"
	case code == 45 || code == 48:
		return "Névoa"
	case code >= 51 && code <= 57:
		return "Garoa"
	case code >= 61 && code <= 67:
		return "Chuva"
	case code >= 71 && code <= 77:
		return "Neve"
	case code >= 80 && code <= 82:
		return "Pancadas"
	case code == 85 || code == 86:
		return "Nevasca"
	case code >= 95 && code <= 99:
		return "Tempestade"
	}
	return "—"
}

// wmoIcon maps WMO weather codes to icon names for the renderer.
func wmoIcon(code int) string {
	switch {
	case code == 0:
		return "clear"
	case code == 1, code == 2:
		return "partly"
	case code == 3:
		return "cloudy"
	case code == 45, code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 80 && code <= 82:
		return "showers"
	case code >= 71 && code <= 77, code == 85, code == 86:
		return "snow"
	case code >= 95 && code <= 99:
		return "storm"
	}
	return "cloudy"
}

func weatherCard() Card {
	city := envOr("DASH_CITY", "Blumenau")
	lat := envOr("DASH_LAT", "-26.9194")
	lon := envOr("DASH_LON", "-49.0661")
	tz := envOr("DASH_TZ", "America/Sao_Paulo")

	api := fmt.Sprintf("https://api.open-meteo.com/v1/forecast"+
		"?latitude=%s&longitude=%s"+
		"&current=temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m"+
		"&daily=temperature_2m_max,temperature_2m_min"+
		"&timezone=%s&forecast_days=1",
		url.QueryEscape(lat), url.QueryEscape(lon), url.QueryEscape(tz))

	title := "Tempo · " + city
	unavailable := Card{Title: title, Rows: []Row{{"", "Indisponível"}}}

	body, err := fetchGET(api)
	if err != nil {
		return unavailable
	}
	var w openMeteoResp
	if err := json.Unmarshal(body, &w); err != nil {
		return unavailable
	}

	rows := []Row{}
	if len(w.Daily.TempMax) > 0 {
		rows = append(rows, Row{"Máx", fmt.Sprintf("%.0f°C", w.Daily.TempMax[0])})
	}
	if len(w.Daily.TempMin) > 0 {
		rows = append(rows, Row{"Mín", fmt.Sprintf("%.0f°C", w.Daily.TempMin[0])})
	}
	rows = append(rows,
		Row{"Umidade", fmt.Sprintf("%d%%", w.Current.Humidity)},
		Row{"Vento", fmt.Sprintf("%.0f km/h", w.Current.WindSpeed)},
	)
	return Card{
		Title: title,
		Icon:  wmoIcon(w.Current.WeatherCode),
		Hero:  fmt.Sprintf("%.0f°C", w.Current.Temperature),
		Sub:   fmt.Sprintf("%s · sensação %.0f°C", wmoText(w.Current.WeatherCode), w.Current.Apparent),
		Rows:  rows,
	}
}

// ---------- Markets (Yahoo Finance, no API key) ----------

// indexQuotes are always shown, in this order, at the top of the card.
var indexQuotes = []struct {
	symbol string
	label  string
	dec    int
	prefix string
}{
	{"^BVSP", "IBOV", 0, ""},
	{"BRL=X", "Dólar", 2, "R$ "},
	{"^GSPC", "S&P 500", 0, "US$ "},
}

// defaultQuotes is the watchlist; the card shows its biggest movers.
var defaultQuotes = []struct {
	symbol string
	label  string
	dec    int
	prefix string
}{
	// Brazilian stocks.
	{"AGRO3.SA", "AGRO3", 2, "R$ "},
	{"BBAS3.SA", "BBAS3", 2, "R$ "},
	{"BBDC4.SA", "BBDC4", 2, "R$ "},
	{"BPAC11.SA", "BPAC11", 2, "R$ "},
	{"CSAN3.SA", "CSAN3", 2, "R$ "},
	{"EGIE3.SA", "EGIE3", 2, "R$ "},
	{"ITUB4.SA", "ITUB4", 2, "R$ "},
	{"KLBN11.SA", "KLBN11", 2, "R$ "},
	{"MDIA3.SA", "MDIA3", 2, "R$ "},
	{"RADL3.SA", "RADL3", 2, "R$ "},
	{"RDOR3.SA", "RDOR3", 2, "R$ "},
	{"VALE3.SA", "VALE3", 2, "R$ "},
	{"WEGE3.SA", "WEGE3", 2, "R$ "},
	// Equity ETF and fixed-income ETF.
	{"AUVP11.SA", "AUVP11", 2, "R$ "},
	{"AUPO11.SA", "AUPO11", 2, "R$ "},
	// Real-estate funds (logistics, malls) and real-estate credit fund.
	{"BTLG11.SA", "BTLG11", 2, "R$ "},
	{"XPML11.SA", "XPML11", 2, "R$ "},
	{"RECR11.SA", "RECR11", 2, "R$ "},
	// Dollar funds.
	{"QQQ", "QQQ", 2, "US$ "},
	{"AOK", "AOK", 2, "US$ "},
	{"SPHQ", "SPHQ", 2, "US$ "},
}

type yahooChart struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				PreviousClose      float64 `json:"previousClose"`
			} `json:"meta"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

// yahooQuote returns the latest price and previous close for a symbol.
func yahooQuote(symbol string) (price, prev float64, err error) {
	api := "https://query1.finance.yahoo.com/v8/finance/chart/" +
		url.QueryEscape(symbol) + "?range=1d&interval=1d"
	body, err := fetchGET(api)
	if err != nil {
		return 0, 0, err
	}
	var c yahooChart
	if err := json.Unmarshal(body, &c); err != nil {
		return 0, 0, err
	}
	if c.Chart.Error != nil || len(c.Chart.Result) == 0 {
		return 0, 0, fmt.Errorf("no data for %s", symbol)
	}
	m := c.Chart.Result[0].Meta
	prev = m.ChartPreviousClose
	if prev == 0 {
		prev = m.PreviousClose
	}
	return m.RegularMarketPrice, prev, nil
}

// ptNum formats a number with Brazilian grouping (1.234,56).
func ptNum(v float64, dec int) string {
	neg := v < 0
	s := strconv.FormatFloat(math.Abs(v), 'f', dec, 64)
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if len(parts) == 2 {
		b.WriteByte(',')
		b.WriteString(parts[1])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// formatQuote renders "R$ 38,72 ▲ +2,4%" style values.
func formatQuote(prefix string, price float64, dec int, chg float64) string {
	arrow, sign := "▲", "+"
	if chg < 0 {
		arrow, sign = "▼", "−"
	}
	return fmt.Sprintf("%s%s %s %s%s%%",
		prefix, ptNum(price, dec), arrow, sign, ptNum(math.Abs(chg), 1))
}

func marketsCard() Card {
	var rows []Row

	// Indices first, always shown.
	for _, q := range indexQuotes {
		price, prev, err := yahooQuote(q.symbol)
		if err != nil {
			rows = append(rows, Row{q.label, "—"})
			continue
		}
		chg := 0.0
		if prev > 0 {
			chg = (price - prev) / prev * 100
		}
		rows = append(rows, Row{q.label, formatQuote(q.prefix, price, q.dec, chg)})
		time.Sleep(150 * time.Millisecond)
	}

	// Watchlist: keep only the biggest movers, ranked.
	type mover struct {
		label  string
		price  float64
		chg    float64
		dec    int
		prefix string
	}
	var movers []mover
	for _, q := range defaultQuotes {
		price, prev, err := yahooQuote(q.symbol)
		if err != nil {
			continue
		}
		chg := 0.0
		if prev > 0 {
			chg = (price - prev) / prev * 100
		}
		movers = append(movers, mover{q.label, price, chg, q.dec, q.prefix})
		time.Sleep(150 * time.Millisecond)
	}
	sort.Slice(movers, func(i, j int) bool { return movers[i].chg > movers[j].chg })

	var gains, losses []mover
	for _, m := range movers {
		if m.chg > 0 && len(gains) < 3 {
			gains = append(gains, m)
		}
	}
	for i := len(movers) - 1; i >= 0; i-- {
		if movers[i].chg < 0 && len(losses) < 3 {
			losses = append(losses, movers[i])
		}
	}
	if len(gains) > 0 {
		rows = append(rows, Row{"", "Altas"})
		for _, m := range gains {
			rows = append(rows, Row{m.label, formatQuote(m.prefix, m.price, m.dec, m.chg)})
		}
	}
	if len(losses) > 0 {
		rows = append(rows, Row{"", "Baixas"})
		for _, m := range losses {
			rows = append(rows, Row{m.label, formatQuote(m.prefix, m.price, m.dec, m.chg)})
		}
	}

	if len(rows) == 0 {
		return Card{Title: "Mercados", Rows: []Row{{"", "Indisponível"}}}
	}
	return Card{Title: "Mercados", Rows: rows}
}

// buildCards assembles every card, top to bottom, for the dashboard.
func buildCards() []Card {
	return []Card{
		clockCard(),
		weatherCard(),
		marketsCard(),
	}
}
