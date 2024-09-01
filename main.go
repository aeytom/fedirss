package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aeytom/fedirss/app"
	"github.com/mattn/go-mastodon"
	"github.com/mmcdole/gofeed"
)

const UserAgent = "fedirss/0.1"

var (
	tags = []string{
		"Berlin",
		"Charlottenburg",
		"Festival",
		"Filmfestival",
		"Filmfestspiele",
		"Friedrichshain",
		"Hellersdorf",
		"Köpenick",
		"Kreuzberg",
		"Lichtenberg",
		"Marzahn",
		"Museum",
		"Neukölln",
		"Pankow",
		"Polizei",
		"Reinickendorf",
		"Schöneberg",
		"Spandau",
		"Steglitz",
		"Tempelhof",
		"Theater",
		"Treptow",
		"Wilmersdorf",
		"Wochenende",
		"Zehlendorf",
	}
	tagsRe *regexp.Regexp
)

func main() {

	settings := app.LoadConfig()
	mc := settings.GetClient()

	url := settings.Feed.Url
	tagsRe = regexp.MustCompile(`\b(` + strings.Join(tags, "|") + `)\b`)

	fp := gofeed.NewParser()
	fp.UserAgent = UserAgent
	resp, err := fp.ParseURL(url)
	if err != nil {
		settings.Fatal(err)
	}

	fmt.Println(resp.Title)

	defer settings.CloseDatabase()

	for _, item := range resp.Items {
		fmt.Println(item.PublishedParsed.Format(time.RFC3339) + " " + item.Title + " " + item.Link)
		if !settings.StoreItem(item) {
			settings.Log("… not stored")
		}
	}

	for item := settings.GetUnsent(); item != nil; item = settings.GetUnsent() {
		scheduledAt := item.PublishedParsed
		title := hashtag(item.Title)
		status := title + "\n\n" + item.Description + "\n\n" + strings.Join(item.Categories, " ")
		length := mblen(hashtag(status))
		if length > 485 {
			status = hashtag(left(status, 485)) + "…"
		}
		toot := &mastodon.Toot{
			Status:      status + "\n" + item.Link,
			Sensitive:   false,
			Visibility:  "public",
			Language:    "de",
			ScheduledAt: scheduledAt,
		}
		if item.Image != nil {
			if img, err := ImageHttp(item.Image.URL); err == nil {
				defer img.Close()
				if a, err := mc.UploadMediaFromReader(context.Background(), img); err != nil {
					settings.Log(err)
				} else {
					a.Description = item.Image.Title
					settings.Log("media: ", a)
					toot.MediaIDs = append(toot.MediaIDs, a.ID)
				}
			} else {
				settings.Log(err)
			}
		}
		if _, err := mc.PostStatus(context.Background(), toot); err != nil {
			settings.Logf("%s – %s – (%d/%d) :: %s", title, status, mblen(title), mblen(status), err.Error())
			settings.MarkError(item, err)
			continue
		} else {
			settings.MarkSent(item)
			settings.Log("… sent ", item.Link)
			return
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

// Image gets a image with HTTP
func ImageHttp(url string) (io.ReadCloser, error) {
	log.Print(url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Accept", "image/jpeg")
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return nil, err
	} else if resp.StatusCode != 200 {
		return nil, errors.New("invalid status code " + fmt.Sprint(resp.StatusCode))
	} else if resp.Header.Get("content-type") != "image/jpeg" {
		return nil, errors.New("invalid content type " + resp.Header.Get("content-type"))
	} else {
		return resp.Body, nil
	}
}
