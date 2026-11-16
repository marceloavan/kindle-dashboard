package main

import (
	"os"
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

// currentTime returns the current time in the dashboard's display timezone
// (DASH_TZ, an IANA name such as "Europe/Lisbon"), defaulting to UTC.
func currentTime() time.Time {
	name := os.Getenv("DASH_TZ")
	if name == "" {
		name = "UTC"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

func clockCard() Card {
	now := currentTime()
	zone := os.Getenv("DASH_TZ")
	if zone == "" {
		zone = "UTC"
	}
	return Card{
		Title: "Clock",
		Rows: []Row{
			{"Time", now.Format("15:04")},
			{"Date", now.Format("2006-01-02")},
			{"Day", now.Format("Monday")},
			{"Zone", zone},
		},
	}
}

func weatherCard() Card {
	// SAMPLE data — replace with a real weather API.
	return Card{
		Title: "Weather (sample)",
		Rows: []Row{
			{"Now", "18°C  ·  clear"},
			{"High", "21°C"},
			{"Low", "11°C"},
			{"Humidity", "52%"},
			{"Wind", "12 km/h NW"},
		},
	}
}

func calendarCard() Card {
	// SAMPLE data — replace with a real calendar feed.
	return Card{
		Title: "Today (sample)",
		Rows: []Row{
			{"09:00", "Stand-up"},
			{"11:30", "1:1 with Sam"},
			{"14:00", "Deep work"},
			{"17:30", "Gym"},
		},
	}
}

func notesCard() Card {
	// SAMPLE data — replace with headlines, todos, quotes, etc.
	return Card{
		Title: "Notes (sample)",
		Rows: []Row{
			{"", "Ship the dashboard prototype."},
			{"", "Confirm Kindle serial prefix."},
			{"", "Do NOT update firmware past 5.16.2.1.1."},
		},
	}
}

// buildCards assembles every card, top to bottom, for the dashboard.
func buildCards() []Card {
	return []Card{
		clockCard(),
		weatherCard(),
		calendarCard(),
		notesCard(),
	}
}
