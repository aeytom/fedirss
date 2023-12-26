package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aeytom/fedilpd/app"
	"github.com/mattn/go-mastodon"
	"github.com/ungerik/go-rss"
)

var (
	tags = []string{
		"Berlin",
		"Polizei",
		"Friedrichshain",
		"Kreuzberg",
		"Pankow",
		"Charlottenburg",
		"Wilmersdorf",
		"Spandau",
		"Steglitz",
		"Zehlendorf",
		"Tempelhof",
		"Schöneberg",
		"Neukölln",
		"Treptow",
		"Köpenick",
		"Marzahn",
		"Hellersdorf",
		"Lichtenberg",
		"Reinickendorf",
	}
	tagsRe *regexp.Regexp
)

func main() {

	settings := app.LoadConfig()
	mc := settings.GetClient()

	t := time.Now().AddDate(0, 0, -1)
	url := settings.Feed.Url + t.Format("02.01.2006")

	resp, err := rss.Read(url, true)
	if err != nil {
		settings.Fatal(err)
	}

	channel, err := rss.Regular(resp)
	if err != nil {
		fmt.Println(err)
	}

	tagsRe = regexp.MustCompile(`(?i)\b(` + strings.Join(tags, "|") + `)`)
	fmt.Println(channel.Title)

	defer settings.CloseDatabase()

	for _, item := range channel.Item {
		time, err := item.PubDate.Parse()
		if err != nil {
			settings.Log(err)
			continue
		}
		fmt.Println(time.String() + " " + item.Title + " " + item.Link)
		if !settings.StoreItem(&item) {
			break
		}
	}

	for item := settings.GetUnsent(); item != nil; item = settings.GetUnsent() {
		scheduledAt, err := item.PubDate.Parse()
		if err != nil {
			settings.Log(err)
			continue
		}
		link := regexp.MustCompile(`^.*\.(\d+)\.php$`).ReplaceAllString(item.Link, "https://berlin.de/-ii$1")
		categories := hashtag(strings.Join(item.Category, " "))
		status := hashtag(item.Description) + "\n\n" + categories + "\n" + link
		length := len(item.Title + status)
		if length > 500 {
			status = hashtag(item.Description[:len(item.Description)-(length-501)]) + "…\n\n" + categories + "\n" + link
		}
		toot := &mastodon.Toot{
			Status:      status,
			Sensitive:   true,
			SpoilerText: item.Title,
			Visibility:  "public",
			Language:    "de",
			ScheduledAt: &scheduledAt,
			Poll:        &mastodon.TootPoll{},
		}
		if _, err := mc.PostStatus(context.Background(), toot); err != nil {
			settings.MarkError(item, err)
			settings.Log(err)
			continue
		} else {
			settings.MarkSent(item)
			settings.Log("… sent ", item.Link)
		}
	}
}

func hashtag(text string) string {
	out := tagsRe.ReplaceAllString(text, "#$1")
	return out
}
