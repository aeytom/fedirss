package main

import (
	"context"
	"fmt"
	"html"
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

	tagsRe = regexp.MustCompile(`\b(` + strings.Join(tags, "|") + `)\b`)
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
		title := hashtag(item.Title)
		link := regexp.MustCompile(`^.*\.(\d+)\.php$`).ReplaceAllString(item.Link, "https://berlin.de/-ii$1")
		footer := "\n\n" + hashtag(strings.Join(item.Category, " ")) + "\n" + link
		status := hashtag(item.Description) + footer
		length := mblen(title + status)
		if length > 500 {
			status = hashtag(left(item.Description, 499-mblen(title)-mblen(footer))) + "…" + footer
		}
		toot := &mastodon.Toot{
			Status:      status,
			Sensitive:   true,
			SpoilerText: title,
			Visibility:  "public",
			Language:    "de",
			ScheduledAt: &scheduledAt,
		}
		if _, err := mc.PostStatus(context.Background(), toot); err != nil {
			settings.Logf("%s – %s – (%d/%d) :: %s", title, status, mblen(title), mblen(status), err.Error())
			settings.MarkError(item, err)
			continue
		} else {
			settings.MarkSent(item)
			settings.Log("… sent ", item.Link)
		}
	}
}

func hashtag(text string) string {
	out := tagsRe.ReplaceAllString(html.UnescapeString(text), "#$1")
	return out
}

func mblen(text string) int {
	return len([]rune(text))
}

func left(input string, length int) string {
	asRunes := []rune(input)
	return string(asRunes[0 : length-1])
}
