package main

import "github.com/gocolly/colly/v2"
import "log"
import "flag"
import "strings"
import "os"
import "net/http"
import "io"
import "path/filepath"
import "image"
import _ "image/gif"
import _ "image/jpeg"
import _ "image/png"
import _ "golang.org/x/image/webp"
import "bytes"
import "encoding/json"
import "sync"

func main() {
	var url string
	var p string
	var c int

	flag.StringVar(&url, "url", "", "URL for scraping")
	flag.StringVar(&p, "p", "", "File path written in urls")
	flag.IntVar(&c, "c", 3, "Concurrency")
	flag.Parse()

	if url != "" {
		doScraping(url)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		log.Fatalln("Fail to opne file,", p)
	}

	var urls []string
	if err = json.Unmarshal(data, &urls); err != nil {
		log.Fatalln("Fail to decode file to path array")
	}
	var wg sync.WaitGroup
	guard := make(chan struct{}, c)
	for i := 0; i < len(urls); i += 1 {
		guard <- struct{}{}
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			doScraping(urls[index])
			<-guard
		}(i)
	}
	wg.Wait()
}

func doScraping(url string) {
	dir := url[strings.LastIndex(strings.TrimRight(url, "/"), "/")+1:]
	err := os.Mkdir(dir, 0755)
	if err != nil {
		if os.IsExist(err) {
			log.Println("Directory is already exists,", dir)
			return
		}
		log.Fatalln("Fail to create directory,", dir, err)
	}

	c := colly.NewCollector()

	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src == "" {
			return
		}
		width := e.Attr("width")
		height := e.Attr("height")
		if width == "" && height == "" {
			return
		}
	fetch:
		resp, err := http.Get(src)
		if err != nil {
			log.Fatalln("Fail to get image,", err)
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			log.Println("Request is malformed")
			return
		}
		if resp.StatusCode >= 500 {
			log.Println("Server error, retry")
			goto fetch
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln("Fail to read body,", err)
		}
		resp.Body.Close()
		if _, _, err = image.Decode(bytes.NewReader(body)); err != nil {
			log.Println("Image is broken, retry")
			goto fetch
		}
		filename := filepath.Join(dir, src[strings.LastIndex(src, "/"):])
		file, err := os.Create(filename)
		if err != nil {
			log.Fatalln("Fail to create file,", filename)
		}
		defer file.Close()
		if _, err := file.Write(body); err != nil {
			log.Fatalln("Fail to write image,", err)
		}
		log.Println("Save image to", filename)
	})

	log.Println("Start to do scraping,", url)
	c.Visit(url)
}
