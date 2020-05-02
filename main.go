// scrapedestroyallsoftwarevideos project main.go
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	//"github.com/robertkrimen/otto"
)

type task interface {
	process()
	print()
}

type factory interface {
	make(line string) task
}

type springerEbook struct {
	book_url      string
	book_name     string
	book_pdf_url  string
	book_epub_url string

	err                  error
	book_download_status string
}

type bookfactory struct {
}

func (*bookfactory) make(line string) task {
	return &springerEbook{book_url: strings.Split(line, "~")[1], book_name: strings.Split(line, "~")[0]}
}

func (book *springerEbook) process() {

	pdf_url, epub_url, err := GetBookStoreUrl(book.book_url)
	if err != nil {
		book.err = err
		return
	}

	if pdf_url == "" {

		book.err = errors.New("link to the pdf not present")
		return

	}

	book.book_pdf_url = pdf_url
	book.book_epub_url = epub_url

	book.store()

}

func (book *springerEbook) store() {
	storage_path := "books/"

	cli := &http.Client{}
	res, err := cli.Get(book.book_pdf_url)
	if err != nil {
		book.err = err
		return
	}

	f, err := os.Create(storage_path + book.book_name + ".pdf")
	if err != nil {

		book.err = err
		return
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		book.err = err
		return
	}
	err = res.Body.Close()
	if err != nil {
		book.err = err
		return
	}
	err = f.Close()
	if err != nil {
		book.err = err
		return
	}
	book.book_download_status = "complete"

}

func (book *springerEbook) print() {
	if book.err != nil {
		fmt.Printf("Error occured for book:%s and error is:%s\n", book.book_url, book.err)
		return
	}

	fmt.Println(book.book_name, ":", book.book_url, book.book_pdf_url, book.book_download_status)

}

func run(f factory) {
	var wg sync.WaitGroup
	in := make(chan task)
	wg.Add(1)

	go func() {

		for _, line := range GetBooksInfo() {
			in <- f.make(line)

		}
		close(in)
		wg.Done()

	}()

	out := make(chan task)

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			for t := range in {
				t.process()
				out <- t
			}
			wg.Done()
		}()

	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for t := range out {
		t.print()
	}

}

func main() {

	run(&bookfactory{})

}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func GetBookStoreUrl(url string) (string, string, error) {

	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", "", err
	}
	var pdf_url string
	var epub_url string
	var pdf_link_present bool

	doc.Find(".cta-button-container--stacked > div:nth-child(1) > a:nth-child(1)").EachWithBreak(func(index int, item *goquery.Selection) bool {

		pdf_url, pdf_link_present = item.Attr("href")

		return false
	})

	if !pdf_link_present {

		doc.Find(".cta-button-container__item > div:nth-child(1) > a:nth-child(1)").EachWithBreak(func(index int, item *goquery.Selection) bool {

			pdf_url, pdf_link_present = item.Attr("href")

			return false
		})
	}

	doc.Find(".cta-button-container--stacked > div:nth-child(2) > a:nth-child(1)").EachWithBreak(func(index int, item *goquery.Selection) bool {

		epub_url, _ = item.Attr("href")
		return false
	})

	baseUrl := "https://link.springer.com"

	if pdf_link_present {
		pdf_url = baseUrl + pdf_url
	}

	epub_url = baseUrl + epub_url

	return pdf_url, epub_url, nil

}

func GetBooksInfo() []string {
	return booksInfo

}
