package main

import (
	"fmt"
	htmlTemplate "html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/gomarkdown/markdown"
)

type BlogEntry struct {
	Name    string
	Path    string
	Content []byte

	Title       string
	Date        time.Time
	Slug        string
	Description string
	Thumbnail   string
}

type BlogEntryView struct {
	Title string
	Slug  string
}

type PostTemplateData struct {
	PrevEntryId int
	EntryId     int
	NextEntryId int
	Entry       BlogEntry
	Entries     []BlogEntry
	Date        string
	Content     htmlTemplate.HTML
}

type RSSEntry struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	Guid        string
}

const static_dir = "static/"
const bin_dir = "docs/"
const in_time_fmt = "01-02-2006 15:04 MST"
const out_time_fmt = "Mon, 02 Jan 2006"
const rss_time_fmt = "Mon, 02 Jan 2006 15:04:05 -0700"


const templates_dir = "templates/"
const blog_post_template_name = "blog_post.html"
const rss_template_name = "rss.xml"

func generate_slug(e BlogEntry) string {
	return "blog/" + fmt.Sprintf("%s", e.Slug)
}

func generate_redirect(bin_name string, to string) {
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	redirect_str := fmt.Sprintf("<meta name=\"color-scheme\" content=\"light dark\"><meta http-equiv=\"refresh\" content=\"0; URL=%s\" />", to)
	f.WriteString(redirect_str)
}

func generate_posts(entries []BlogEntry, html_templpath string) {
	funcMap := textTemplate.FuncMap{
		"generateSlug": generate_slug,
	}

	tmpl, err := htmlTemplate.New(blog_post_template_name).Funcs(funcMap).ParseFiles(html_templpath)
	if err != nil {
		panic(err)
	}

	for i, entry := range entries {
		// generate redirect page
		if i == 0 {
			slug_link := fmt.Sprintf("%s", entry.Slug)
			headname := fmt.Sprintf("%sindex.html", bin_dir)
			generate_redirect(headname, slug_link)
		}

		// generate blog post page using template
		bin_name := fmt.Sprintf("%s%s.html", bin_dir, entry.Slug)
		f, err := os.Create(bin_name)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		data := PostTemplateData{
			PrevEntryId: i - 1,
			EntryId:     i,
			NextEntryId: i + 1,
			Entry:       entry,
			Entries:     entries,
			Date:        entry.Date.Format(out_time_fmt),
			Content:     htmlTemplate.HTML(markdown.ToHTML(entry.Content, nil, nil)),
		}

		err = tmpl.Execute(f, data)
		if err != nil {
			panic(err)
		}
	}
}

func generate_rss(entries []BlogEntry, rss_templpath string) {
	tmpl, err := textTemplate.ParseFiles(rss_templpath)
	if err != nil {
		panic(err)
	}

	rssEntries := []RSSEntry{}

	for _, entry := range entries {
		date_str := entry.Date.Format(rss_time_fmt)
		post_link := fmt.Sprintf("https://gravitymoth.com/blog/%s", entry.Slug)
		rssEntry := RSSEntry{
			Title:       entry.Title,
			Description: entry.Description,
			PubDate:     date_str,
			Link:        post_link,
			Guid:        post_link,
		}
		rssEntries = append(rssEntries, rssEntry)
	}

	bin_name := fmt.Sprintf("%sfeed.xml", bin_dir)
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = tmpl.Execute(f, rssEntries)
	if err != nil {
		panic(err)
	}
}

func process_blog_posts() []BlogEntry {
	files, err := ioutil.ReadDir(static_dir)
	if err != nil {
		log.Fatal(err)
	}
	
	mds := make([]BlogEntry, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".blg") {
			staticname := fmt.Sprintf("%s%s", static_dir, file.Name())
			content, err := os.ReadFile(staticname)
			if err != nil {
				log.Fatal(err)
			}

			title := ""
			date_str := ""
			slug := ""
			desc := ""
			img := ""

			keymap := make(map[string]bool)
			keymap["title: "] = false
			keymap["date: "] = false
			keymap["desc: "] = false
			keymap["slug: "] = false
			keymap["img: "] = false

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				for k, _ := range keymap {
					if strings.HasPrefix(line, k) {
						val := line[len(k):]
						switch k {
						case "title: ":
							title = val
						case "date: ":
							date_str = val
						case "slug: ":
							slug = val
						case "desc: ":
							desc = val
						case "img: ":
							img = val
						}
					}
				}
			}
			content = content[strings.Index(string(content), "\n\n")+1:]

			date, err := time.Parse(in_time_fmt, string(date_str))
			if err != nil {
				log.Fatal(err)
			}

			// If the post doesn't have a thumbnail, use the default site logo instead
			if img == "" {
				img = "logo_bg.png"
			}

			md := BlogEntry{Name: file.Name(), Path: static_dir, Content: content, Title: string(title), Date: date, Slug: string(slug), Description: string(desc), Thumbnail: string(img)}
			mds = append(mds, md)
		}
	}
	
	return mds
}

func main() {
	// Process templates
	templates, err := ioutil.ReadDir(templates_dir)
	if err != nil {
		log.Fatal(err)
	}
	
	var blog_post_template_file fs.FileInfo = nil
	var rss_template_file fs.FileInfo = nil
	for _, file := range templates {
		switch file.Name() {
		case blog_post_template_name:
			blog_post_template_file = file
		case rss_template_name:
			rss_template_file = file
		}
	}
	
	if blog_post_template_file == nil {
		log.Fatal("Couldn't find blog-post template!\n")
	}
	if rss_template_file == nil {
		log.Fatal("Couldn't find rss template!\n")
	}
	
	blog_post_templpath := fmt.Sprintf("%s%s", templates_dir, blog_post_template_file.Name())
	rss_templpath := fmt.Sprintf("%s%s", templates_dir, rss_template_file.Name())
	
	// Process blog posts
	mds := process_blog_posts()

	_ = os.RemoveAll(bin_dir)
	_ = os.Mkdir(bin_dir, os.ModePerm)

	sort.SliceStable(mds, func(i, j int) bool {
		return mds[i].Date.After(mds[j].Date)
	})

	generate_posts(mds, blog_post_templpath)
	generate_rss(mds, rss_templpath)

	fmt.Printf("site generated\n")
}
