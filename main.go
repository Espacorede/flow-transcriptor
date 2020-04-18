package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	configCsv, err := readCsv("config.csv")
	if err != nil {
		log.Fatalln(err)
	}

	// argv := os.Args[1:]

	botUsername := configCsv[0]
	botPassword := configCsv[1]

	bot := wiki(botUsername, botPassword)

	namespaces := []int{1, 3, 5, 7, 9, 11, 13, 15, 829}

	var flowPages []string

	log.Printf("Login OK! %s - %s\n", bot.username, bot.token)

	for _, ns := range namespaces {
		log.Printf("Getting namespace %d\n", ns)
		gap := bot.getAllPages(ns)
		if len(gap) == 0 {
			log.Printf("This namespace doesn't seem to have any flow pages at all :o")
		} else {
			flowPages = append(flowPages, gap...)
		}
	}

	dirName := "transcripts"
	_, err = os.Stat(dirName)

	if os.IsNotExist(err) {
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Found a total of %d flow-board pages\n", len(flowPages))

	count := 0

	for _, page := range flowPages {
		println(page)
		format := bot.formatFlow(page)

		if format != "" {
			log.Printf("Formatting page %s\n", page)
			filename := fmt.Sprintf("%s/%s.txt", dirName, safeFileName(page))

			f, err := os.Create(filename)

			if err != nil {
				log.Printf("Error creating file %s.txt: %s\n", page, err)
			}

			_, err = f.WriteString(format)

			count++
		}
	}

	log.Printf("We're done, after generating %d pages.", count)
}
