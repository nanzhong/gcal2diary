package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/nanzhong/gcal2diary"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	flagCredentials string
	flagToken       string
	flagDebug       bool
	flagDateStyle   string
)

func init() {
	var tokenPath string
	configDir, _ := os.UserConfigDir()
	if configDir != "" {
		tokenPath = configDir + "/gcal2diary/token.json"
	} else {
		tokenPath = "./token.json"
	}

	flag.StringVar(&flagCredentials, "credentials", "./credentials.json", "Path to the credentials file.")
	flag.StringVar(&flagToken, "token", tokenPath, "Path to the oauth token to use.")
	flag.BoolVar(&flagDebug, "debug", false, "Print debug information to stderr.")
	flag.StringVar(&flagDateStyle, "date-style", "iso", "Date style to use (iso, us, eu).")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	log := log.New(os.Stderr, "", 0)

	var dateStyle gcal2diary.DateStyle
	switch strings.ToLower(flagDateStyle) {
	case "iso":
		dateStyle = gcal2diary.DateStyleISO
	case "us":
		dateStyle = gcal2diary.DateStyleUS
	case "eu":
		dateStyle = gcal2diary.DateStyleEU
	default:
		log.Fatalf("Invalid date style %s. iso, us, and eu are supported", flagDateStyle)
	}
	writer := gcal2diary.NewDiaryWriter(os.Stdout, dateStyle)

	if flagCredentials == "" {
		log.Fatal("--credentials not set. Must provide path to credentials file")
	}

	credBytes, err := os.ReadFile(flagCredentials)
	if err != nil {
		log.Fatalf("Failed to read credentials file (%s): %s", flagCredentials, err)
	}

	config, err := google.ConfigFromJSON(credBytes, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Failed to parse credentials file (%s): %s", flagCredentials, err)
	}

	token, err := gcal2diary.TokenFromFile(flagToken)
	if err != nil {
		if flagDebug {
			log.Printf("Could not reuse existing auth token: %s", err)
		}

		token, err = gcal2diary.NewTokenFromWeb(ctx, config)
		if err != nil {
			log.Fatalf("Failed to auth: %s", err)
		}

		if flagDebug {
			log.Printf("Saving credential file to: %s\n", flagToken)
		}
		err := gcal2diary.SaveToken(flagToken, token)
		if err != nil {
			log.Printf("Failed to save auth token for reuse: %s", err)
		}
	}

	client := config.Client(ctx, token)
	calSvc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to construct gcal client: %v", err)
	}

	now := time.Now()
	tMin := now.Add(-31 * 24 * time.Hour)
	tMax := now.Add(31 * 24 * time.Hour)

	var pageToken string
	for {
		events, err := calSvc.Events.List("primary").
			TimeMin(tMin.Format(time.RFC3339)).
			TimeMax(tMax.Format(time.RFC3339)).
			ShowDeleted(false).
			SingleEvents(true).
			PageToken(pageToken).
			OrderBy("startTime").
			Do()
		if err != nil {
			log.Fatalf("Failed to retrieve events: %s", err)
		}

		for _, e := range events.Items {
			if flagDebug {
				json.NewEncoder(os.Stderr).Encode(e)
			}
			writer.Write(e)
		}

		if events.NextPageToken == "" {
			break
		}
		pageToken = events.NextPageToken
	}
}
