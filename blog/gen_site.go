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

type SiteTemplate struct {
	File fs.FileInfo
	Path string
}

type SiteTemplates struct {
	Index    SiteTemplate
	BlogPost SiteTemplate
	RSS      SiteTemplate
}

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
	Slug        string
	Description string
	PubDate     string
	Guid        string
}

const static_dir = "static/"
const bin_dir = "docs/"
const bin2_dir = "pages/"
const in_time_fmt = "01-02-2006 15:04 MST"
const out_time_fmt = "Mon, 02 Jan 2006"
const rss_time_fmt = "Mon, 02 Jan 2006 15:04:05 -0700"

const templates_dir = "templates/"
const index_template_name = "index.html"
const blog_post_template_name = "blog_post.html"
const rss_template_name = "rss.xml"

func generate_slug(e BlogEntry) string {
	return "blog/" + fmt.Sprintf("%s", e.Slug)
}

type FutureProject struct {
	Title       string
	Image       string
	ImageAlt    string
	Icon        string
	Description htmlTemplate.HTML
}

type HomeData struct {
	BlogPosts []RSSEntry
	Projects  []FutureProject
}

func generate_pages(entries []BlogEntry, templ SiteTemplate) {
	funcMap := textTemplate.FuncMap{
		"generateSlug": generate_slug,
	}

	tmpl, err := htmlTemplate.New(index_template_name).Funcs(funcMap).ParseFiles(templ.Path)
	if err != nil {
		panic(err)
	}

	rssEntries := []RSSEntry{}

	for _, entry := range entries {
		date_str := entry.Date.Format(out_time_fmt)
		rssEntry := RSSEntry{
			Title:       entry.Title,
			Description: entry.Description,
			PubDate:     date_str,
			Slug:        generate_slug(entry),
		}
		rssEntries = append(rssEntries, rssEntry)

		if len(rssEntries) >= 3 {
			break
		}
	}

	futureProjects := []FutureProject{
		{
			Title:    "Debugging Should Be Legible",
			Icon:     "fa-solid fa-file-pen",
			Image:    "media/asm.png",
			ImageAlt: "Highlighted Assembly",
			Description: htmlTemplate.HTML(`<p>Why don't we have <b>good syntax highlighting for assembly</b>?</p>
								<p>We use syntax highlighting in many higher-level
								languages daily, to make it easier to <b>quickly scan</b> and <b>follow code flow</b>. It's time to bring disassemblers
								up to speed with modern, 80s technology. Register motion matters, when things go wrong, <b>a little color
								helps you figure out what happened, <i>fast</i></b>.</p>`),
		},
		{
			Title:    "History Matters",
			Icon:     "fa-solid fa-clock-rotate-left",
			Image:    "media/history.png",
			ImageAlt: "Highlighted Emulator State",
			Description: htmlTemplate.HTML(`<p>When dissecting real catastrophes, inspectors build out a timeline and attempt to <b>retrace footsteps</b>.
								Debugging code should work the same way.</p>
								<p><b>Track</b> how your program changes registers and memory
								as it runs, <b>save all your syscalls</b>, and <b>replay your code</b> exactly the way it failed,
								so you can reliably <b>walk backwards</b> from the problem to find the cause.</p>`),
		},
		{
			Title:    "Data Has A Shape",
			Icon:     "fa-solid fa-shapes",
			Image:    "media/striations.png",
			ImageAlt: "Binary as Image",
			Description: htmlTemplate.HTML(`<p><b>Real data has striations, character, and personality.</b> Really understanding a problem might require <b>a
								new way of looking</b> at your code or your dataset. </p>
								<p>We provide the tools to <b>visualize your problem</b>
								in a handful of new, unique ways, to help you <b>build strong intuition</b> about your problems. This is a real
								ELF binary with large static tables of content, plainly visible when displayed
								as a greyscale image.</p>`),
		},
	}

	data := HomeData{
		Projects:  futureProjects,
		BlogPosts: rssEntries,
	}

	bin_name := fmt.Sprintf("%sindex.html", bin2_dir)
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = tmpl.Execute(f, data)
	if err != nil {
		panic(err)
	}
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

func generate_posts(entries []BlogEntry, templ SiteTemplate) {
	funcMap := textTemplate.FuncMap{
		"generateSlug": generate_slug,
	}

	tmpl, err := htmlTemplate.New(blog_post_template_name).Funcs(funcMap).ParseFiles(templ.Path)
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

func generate_rss(entries []BlogEntry, templ SiteTemplate) {
	tmpl, err := textTemplate.ParseFiles(templ.Path)
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

	template_files := SiteTemplates{}
	for _, file := range templates {
		var templ *SiteTemplate
		name := file.Name()
		switch name {
		case index_template_name:
			templ = &template_files.Index
		case blog_post_template_name:
			templ = &template_files.BlogPost
		case rss_template_name:
			templ = &template_files.RSS
		}

		if templ != nil {
			templ.File = file
			templ.Path = fmt.Sprintf("%s%s", templates_dir, name)
		}
	}

	if template_files.Index.File == nil {
		log.Fatal("Couldn't find index template!\n")
	}
	if template_files.BlogPost.File == nil {
		log.Fatal("Couldn't find blog_post template!\n")
	}
	if template_files.RSS.File == nil {
		log.Fatal("Couldn't find rss template!\n")
	}

	// Process blog posts
	mds := process_blog_posts()

	_ = os.RemoveAll(bin_dir)
	_ = os.Mkdir(bin_dir, os.ModePerm)

	_ = os.RemoveAll(bin2_dir)
	_ = os.Mkdir(bin2_dir, os.ModePerm)

	sort.SliceStable(mds, func(i, j int) bool {
		return mds[i].Date.After(mds[j].Date)
	})

	generate_pages(mds, template_files.Index)
	generate_posts(mds, template_files.BlogPost)
	generate_rss(mds, template_files.RSS)

	fmt.Printf("site generated\n")
}
