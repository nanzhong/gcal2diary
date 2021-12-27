package gcal2diary

import (
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
)

type DateStyle int

const (
	DateStyleISO DateStyle = iota
	DateStyleUS
	DateStyleEU
)

type DiaryWriter struct {
	w         io.Writer
	dateStyle DateStyle
}

func NewDiaryWriter(w io.Writer, dateStyle DateStyle) *DiaryWriter {
	return &DiaryWriter{
		w:         w,
		dateStyle: dateStyle,
	}
}

func (w *DiaryWriter) Write(event *calendar.Event) error {
	var s strings.Builder

	start, err := time.Parse(time.RFC3339, event.Start.DateTime)
	if err != nil {
		return fmt.Errorf("parsing event start time %s: %w", event.Start.Date, err)
	}

	end, err := time.Parse(time.RFC3339, event.End.DateTime)
	if err != nil {
		return fmt.Errorf("parsing event end time %s: %w", event.End.Date, err)
	}

	type timeRange struct {
		date  time.Time
		start string
		end   string
	}
	var timeRanges []timeRange
	begin := start
	for {
		bY, bM, bD := begin.Date()
		eY, eM, eD := end.Date()

		beginDate := time.Date(begin.Year(), begin.Month(), begin.Day(), 0, 0, 0, 0, begin.Location())

		if bY == eY && bM == eM && bD == eD {
			timeRanges = append(timeRanges, timeRange{
				date:  beginDate,
				start: w.formatTime(begin),
				end:   w.formatTime(end),
			})
			break
		}

		timeRanges = append(timeRanges, timeRange{
			date:  beginDate,
			start: w.formatTime(begin),
			end:   "24:00",
		})
		begin = time.Date(begin.Year(), begin.Month(), begin.Day()+1, 0, 0, 0, 0, begin.Location())

		if begin.Equal(end) {
			break
		}
	}

	for _, timeRange := range timeRanges {
		s.WriteString(w.formatDate(timeRange.date))
		s.WriteString(" ")
		s.WriteString(timeRange.start + "-" + timeRange.end)
		s.WriteString(" ")
		s.WriteString(event.Summary)
		s.WriteString("\n")

		if event.Location != "" {
			s.WriteString(" Location: ")
			s.WriteString(w.prefixString(event.Location))
			s.WriteString("\n")
		}

		if event.Description != "" {
			s.WriteString(" Description: ")
			s.WriteString(w.prefixString(event.Description))
			s.WriteString("\n")
		}
	}

	w.w.Write([]byte(s.String()))
	return nil
}

func (w *DiaryWriter) formatDate(t time.Time) string {
	switch w.dateStyle {
	case DateStyleISO:
		return t.Format("2006/01/02")
	case DateStyleUS:
		return t.Format("02/01/2006")
	case DateStyleEU:
		return t.Format("01/02/2006")
	default:
		panic("unknown date style")
	}
}

func (w *DiaryWriter) formatTime(t time.Time) string {
	return t.Format("15:04")
}

func (w *DiaryWriter) prefixString(s string) string {
	var sb strings.Builder
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i != 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(line)

		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
