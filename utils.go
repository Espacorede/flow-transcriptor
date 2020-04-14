package main

import (
	"encoding/csv"
	"os"
)

func readCsv(file string) ([]string, error) {
	data, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	csvreader := csv.NewReader(data)

	values, err := csvreader.Read()
	return values, err
}
