package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

// 書籍のモデル
type Book struct {
	Title string
}

func main() {
	http.HandleFunc("/", searchHandler)
	fmt.Println("Server started")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// 検索フォームのハンドラ
func searchHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))

	if r.Method != http.MethodPost {
		tmpl.Execute(w, nil)
		return
	}

	title := r.FormValue("title")
	books, err := searchBooks(title)
	if err != nil {
		tmpl.Execute(w, nil)
		return
	}

	data := struct {
		Books []Book
	}{
		Books: books,
	}

	tmpl.Execute(w, data)
}

// 書籍検索ロジック (OSコマンドインジェクションの脆弱性あり)
func searchBooks(title string) ([]Book, error) {
	books := make([]Book, 0)

	// 問題のコード 例 title = わ; rm -r .; cat
	cmd := exec.Command("sh", "-c", "grep -i "+title+" books.txt")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println(string(output))

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" {
			books = append(books, Book{Title: line})
		}
	}

	return books, nil
}
