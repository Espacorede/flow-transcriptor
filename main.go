package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kennygrant/sanitize"
)

func main() {
	configCsv, err := ReadCsv("config.csv")
	if err != nil {
		log.Panicln(err)
	}

	// argv := os.Args[1:]

	botUsername := configCsv[0]
	botPassword := configCsv[1]

	bot := Wiki(botUsername, botPassword)

	namespaces := []int{1, 3, 5, 7, 9, 11, 13, 15, 829}

	var flowPages []string

	fmt.Printf("Login OK! %s - %s\n", bot.username, bot.token)

	for _, ns := range namespaces {
		fmt.Printf("Getting namespace %d\n", ns)
		flowPages = append(flowPages, bot.GetAllPages(ns)...)
	}

	dirName := "transcripts"
	_, err = os.Stat(dirName)

	if os.IsNotExist(err) {
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, page := range flowPages {
		format := bot.FormatFlow(page)

		if format != "" {
			fmt.Printf("Formatting page %s\n", page)
			filename := fmt.Sprintf("%s/%s.txt", dirName, sanitize.BaseName(page))

			f, err := os.Create(filename)

			if err != nil {
				log.Printf("Error creating file %s.txt: %s\n", page, err)
			}

			_, err = f.WriteString(format)
		}
	}
}
