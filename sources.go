package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	Rows  []Row
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
	resp, err := client.Get(rawurl)
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
		Title: "Relógio",
		Rows: []Row{
			{"Hora", now.Format("15:04")},
			{"Data", now.Format("02/01/2006")},
			{"Dia", weekdaysPT[now.Weekday()]},
			{"Fuso", envOr("DASH_TZ", "America/Sao_Paulo")},
		},
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

	rows := []Row{
		{"Agora", fmt.Sprintf("%.0f°C · %s", w.Current.Temperature, wmoText(w.Current.WeatherCode))},
		{"Sensação", fmt.Sprintf("%.0f°C", w.Current.Apparent)},
	}
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
	return Card{Title: title, Rows: rows}
}

// ---------- News (RSS, no API key) ----------

type rssFeed struct {
	Channel struct {
		Items []struct {
			Title string `xml:"title"`
		} `xml:"item"`
	} `xml:"channel"`
}

func newsCard() Card {
	name := envOr("DASH_RSS_NAME", "G1")
	feed := envOr("DASH_RSS_URL", "https://g1.globo.com/rss/g1/")

	title := "Notícias · " + name
	unavailable := Card{Title: title, Rows: []Row{{"", "Indisponível"}}}

	body, err := fetchGET(feed)
	if err != nil {
		return unavailable
	}
	var r rssFeed
	if err := xml.Unmarshal(body, &r); err != nil {
		return unavailable
	}

	var rows []Row
	for _, it := range r.Channel.Items {
		t := strings.TrimSpace(it.Title)
		if t == "" {
			continue
		}
		// Cap at ~2 rendered lines (body width fits ~46 chars/line).
		if runes := []rune(t); len(runes) > 88 {
			t = string(runes[:88]) + "…"
		}
		rows = append(rows, Row{"", t})
		if len(rows) == 4 {
			break
		}
	}
	if len(rows) == 0 {
		rows = []Row{{"", "Sem manchetes"}}
	}
	return Card{Title: title, Rows: rows}
}

// buildCards assembles every card, top to bottom, for the dashboard.
func buildCards() []Card {
	return []Card{
		clockCard(),
		weatherCard(),
		newsCard(),
	}
}
