package main

import (
    "fmt"
    "net/http"
    "strings"
    "sync"
    "os"
    "bufio"
)

type Table struct {
    data map[string]Info
    mutex sync.Mutex
}

type Info struct {
    ett bool
    en bool
    msg string
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
            ett: false,
            en: true,
            msg: fmt.Sprintf("En %s\n", word),
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
                ett: true,
                en: true,
                msg: fmt.Sprintf("En %s\nEtt %s\nDet beror på kontexten...\n", word, word),
                hits: 0,
            }
            
        } else {
            t[word] = Info{
                ett: true,
                en: false,
                msg: fmt.Sprintf("Ett %s\n", word),
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
    commonBrowsers := []string{"mozilla", "chrome", "safari", "apple"}
    for _, v := range(commonBrowsers) {
        if strings.Contains(strings.ToLower(ua), v) {
            return true
        }
    }
    return false
}


func handleBrowser(w http.ResponseWriter, r *http.Request) {
    if len(r.URL.Path) == 1 {
        // index html
    }
    fmt.Fprintf(w, "Thanks for your mr Browser\n")
}

func handleCurl(w http.ResponseWriter, r *http.Request) {
    if len(r.URL.Path) == 1 {
        fmt.Fprintf(w, "No input given. Try something like this:\n\t%s/stol\n", r.Host)
        return
    }

    word := strings.TrimPrefix(r.URL.Path, "/")
    info, exists := table.Read(word)

    if !exists {
        fmt.Fprintf(w, "Kunde inte hitta substantivet '%s'\n", word)
        return
    }

    fmt.Fprintf(w, info.msg)
   
    go func() {
        table.IncrementHits(word)
    }()

}

func handler(w http.ResponseWriter, r *http.Request) {
    logging(r)
    //payload := lookup(r.Path)
    isBrowser := isBrowser(r.UserAgent())

    if isBrowser {
        // Return payload as pretty html
        handleBrowser(w, r)

    } else {
        // Return payload as json
        handleCurl(w, r)
    }


}

func main() {
    http.HandleFunc("/", handler)

    addr := ":5050"

    fmt.Printf("Global table contains %v items\n", len(table.data))
    fmt.Println("Listening on:", addr)

    err := http.ListenAndServe(addr, nil)

    if err != nil {
        fmt.Println(err)
    }

    fmt.Println("Exiting")
}
