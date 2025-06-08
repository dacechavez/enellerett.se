package main

import (
	"bufio"
	"fmt"
	"html/template"
	"maps"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
)

type Table struct {
	data  map[string]Info
	mutex sync.Mutex
}

type Info struct {
	ett  bool
	en   bool
	msg  string
	hits uint64
}

var table = NewTable("webserver/en.txt", "webserver/ett.txt")

func NewTable(en string, ett string) *Table {
	enf, err := os.Open(en)
	if err != nil {
		fmt.Println("Failed to open:", err)
		os.Exit(1)
	}

	defer enf.Close()

	ettf, err := os.Open(ett)
	if err != nil {
		fmt.Println("Failed to open", err)
		os.Exit(1)
	}

	defer ettf.Close()

	ens := bufio.NewScanner(enf)
	etts := bufio.NewScanner(ettf)

	t := make(map[string]Info)

	for ens.Scan() {
		word := ens.Text()
		t[word] = Info{
			ett:  false,
			en:   true,
			msg:  fmt.Sprintf("En %s\n", word),
			hits: 0,
		}
	}

	if err := ens.Err(); err != nil {
		fmt.Println("Failed to scan:", err)
		os.Exit(1)
	}

	for etts.Scan() {
		word := etts.Text()
		if _, ok := t[word]; ok {
			t[word] = Info{
				ett:  true,
				en:   true,
				msg:  fmt.Sprintf("En eller ett %s beroende på kontext\n", word),
				hits: 0,
			}
		} else {
			t[word] = Info{
				ett:  true,
				en:   false,
				msg:  fmt.Sprintf("Ett %s\n", word),
				hits: 0,
			}
		}
	}

	if err := etts.Err(); err != nil {
		fmt.Println("Failed to scan:", err)
		os.Exit(1)
	}

	return &Table{data: t}
}

func (t *Table) Read(key string) (Info, bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	value, exists := t.data[key]
	return value, exists
}

func (t *Table) IncrementHits(key string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	value, exists := t.data[key]

	if !exists {
		panic("tried to increment an non existant key")
	}

	value.hits += 1

	t.data[key] = value
}

func logging(r *http.Request) {
	fmt.Println("         URI:", r.RequestURI)
	fmt.Println("        Host:", r.Host)
	fmt.Println("        Path:", r.URL.Path)

	for k, v := range r.Header {
		fmt.Println("      Header:", k, v)
	}

	fmt.Println("   UserAgent:", r.UserAgent())
	fmt.Println("  RemoteAddr:", r.RemoteAddr)
	fmt.Println("Query-string:", r.URL.RawQuery)

	fmt.Println()
}

func isBrowser(ua string) bool {
	commonBrowsers := []string{"mozilla", "chrome", "safari", "apple", "webkit"}
	for _, v := range commonBrowsers {
		if strings.Contains(strings.ToLower(ua), v) {
			return true
		}
	}
	return false
}

func handleBrowser(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) == 1 { // root "/"
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.Execute(w, nil)
	} else {
		s := r.FormValue("s")
		fmt.Printf("s: [%s]\n", s)
		res := lookup(s)
		fmt.Fprintf(w, res)
	}
}

func lookup(word string) string {
	clean := strings.ReplaceAll(word, " ", "")
	clean = strings.ToLower(clean)

	if len(clean) == 0 {
		return ""
	}

	info, exists := table.Read(clean)

	if !exists {
		return fmt.Sprintf("Kunde inte hitta substantivet '%s'\n", word)
	}

	go func() {
		table.IncrementHits(clean)
	}()

	return info.msg
}

func handleCurl(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) == 1 {
		fmt.Fprintf(w, "No input given. Try something like this:\n\t%s/stol\n", r.Host)
		return
	}

	s := strings.TrimPrefix(r.URL.Path, "/")
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	res := lookup(s)

	fmt.Fprintf(w, res)
}

func handler(w http.ResponseWriter, r *http.Request) {
	logging(r)

	isBrowser := isBrowser(r.UserAgent())

	if isBrowser {
		// Return payload as pretty html
		handleBrowser(w, r)
	} else {
		// Return payload as json
		handleCurl(w, r)
	}
}

// Responds with a random word from the table
func gameRandom(w http.ResponseWriter, r *http.Request) {
	logging(r)
	words := slices.Collect(maps.Keys(table.data))
	fmt.Fprintf(w, words[rand.Intn(len(words))])
}

func isCorrect(en bool, noun string) bool {
	info, exists := table.Read(noun)

	if !exists {
		panic("somehow noun from game did not exist in table")
	}

	return info.en == en
}

func gameCheck(en bool, noun string) string {
	guess := "en"

	if !en {
		guess = "ett"
	}

	if isCorrect(en, noun) {
		return fmt.Sprintf("&#9989; %s %s<br>", guess, noun)
	} else {
		return fmt.Sprintf("&#10060; %s %s<br>", guess, noun)
	}
}

func gameCheckEn(w http.ResponseWriter, r *http.Request) {
	logging(r)
	s := r.FormValue("randomNoun")
	w.Header().Set("HX-Trigger", "newRandom")
	fmt.Fprintf(w, gameCheck(true, s))
}

func gameCheckEtt(w http.ResponseWriter, r *http.Request) {
	logging(r)
	s := r.FormValue("randomNoun")
	w.Header().Set("HX-Trigger", "newRandom")
	fmt.Fprintf(w, gameCheck(false, s))
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/game/random", gameRandom)
	http.HandleFunc("/game/check/en", gameCheckEn)
	http.HandleFunc("/game/check/ett", gameCheckEtt)

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`User-agent: *
Allow: /
Sitemap: https://enellerett.se/sitemap.xml`))
	})

	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url>
        <loc>https://enellerett.se/</loc>
        <lastmod>2025-06-08</lastmod>
        <changefreq>weekly</changefreq>
        <priority>1.0</priority>
    </url>
</urlset>`))
	})

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg width="32" height="32" viewBox="0 0 32 32" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#667eea;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#764ba2;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="32" height="32" rx="6" fill="url(#grad)"/>
  <text x="16" y="23" font-family="Arial, sans-serif" font-size="20" font-weight="bold" text-anchor="middle" fill="white">e</text>
</svg>`))
	})

	addr := ":6969"

	fmt.Printf("Global table contains %v items\n", len(table.data))
	fmt.Println("Listening on:", addr)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Exiting")
}
