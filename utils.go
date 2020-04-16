package main

import (
	"encoding/csv"
	"os"
	"regexp"
)

var fileRegex = regexp.MustCompile(`[/\\?%*:|"<>.]+`)

func readCsv(file string) ([]string, error) {
	data, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	csvreader := csv.NewReader(data)

	values, err := csvreader.Read()
	return values, err
}

func safeFileName(name string) string {
	return fileRegex.ReplaceAllString(name, "-")
}
