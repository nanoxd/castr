package main

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nanoxd/castr/config"
	"github.com/nanoxd/castr/rss"
)

// MimeTypeMap list all the usable extensions
var mimeTypeMap = map[string]string{
	".mp3": "audio/mpeg",
	".m4a": "audio/m4a",
	".m4b": "audio/m4b",
}

func escapeURL(str string) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func fileInfo(name string) (string, string) {
	for extension, mimeType := range mimeTypeMap {
		if strings.HasSuffix(name, extension) {
			return extension, mimeType
		}
	}

	return "", ""
}

func main() {
	config := config.Config{}
	if err := config.Load("config.toml"); err != nil {
		panic(err)
	}

	fs := http.FileServer(http.Dir(config.Server.FileRoot))
	http.Handle("/", fs)

	http.HandleFunc(config.Server.FeedPath, FeedHandler)

	http.ListenAndServe(config.Server.Listen, nil)
}

func FeedHandler(w http.ResponseWriter, r *http.Request) {
	config := config.Config{}
	if err := config.Load("config.toml"); err != nil {
		panic(err)
	}

	link, err := url.Parse(config.RSS.URL)
	if err != nil {
		panic(err)
	}
	feedURL, err := url.Parse(config.RSS.URL)
	if err != nil {
		panic(err)
	}
	feedURL.Path = config.Server.FeedPath

	feed := rss.RSS{
		XMLXmlnsAtom:   "http://www.w3.org/2005/Atom",
		XMLXmlnsItunes: "http://www.itunes.com/dtds/podcast-1.0.dtd",
		XMLVersion:     "2.0",
	}
	feed.Channel = &rss.Channel{
		Title:       config.RSS.Title,
		Description: config.RSS.Description,
		Link:        link.String(),
		AtomLink: &rss.AtomLink{
			Href: feedURL.String(),
			Rel:  "self",
			Type: "application/rss+xml",
		},
	}

	items := []*rss.Item{}
	err = filepath.Walk(config.Server.FileRoot, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		extension, mime := fileInfo(info.Name())
		if extension == "" || mime == "" {
			return nil
		}

		title := strings.Replace(info.Name(), extension, "", 1)

		pubDate := info.ModTime().Format(time.RFC1123)
		url, err := escapeURL(config.RSS.URL + strings.Replace(path, config.Server.FileRoot, "", 1))
		if err != nil {
			panic(err)
		}
		enclosure := rss.Enclosure{URL: url, Type: mime, Length: info.Size()}
		item := rss.Item{
			Title:     title,
			PubDate:   pubDate,
			Guid:      url,
			Enclosure: &enclosure,
		}

		items = append(items, &item)
		return nil
	})
	if err != nil {
		panic(err)
	}

	sort.Sort(sort.Reverse(rss.ByPubDate(items)))
	feed.Channel.Item = items

	buf, err := xml.MarshalIndent(feed, "", " ")
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/atom+xml")
	w.Header().Set("Last-Modified", items[0].PubDate)
	w.Write([]byte(xml.Header))
	w.Write(buf)
}
