package output

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type CSVOutput struct {
	fileName string
	log      models.Logger
}

func NewCSVOutput(fileName string, logger models.Logger) *CSVOutput {
	return &CSVOutput{
		fileName: fileName,
		log: logger,
	}
}

func (co *CSVOutput) Write(result models.Result) error {
	file, err := os.Create(co.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	data := [][]string{
		result.Header,
	}
	for _, row := range result.Rows {
		var csvRow []string
		for _, rec := range row {
			csvRow = append(csvRow, fmt.Sprint(rec))
		}
		data = append(data, csvRow)
	}

	w := csv.NewWriter(file)
	err = w.WriteAll(data)
	if err != nil {
		return err
	}
	if err := w.Error(); err != nil {
		return err
	}
	co.log.Info("successfully saved csv to " + co.fileName)
	return nil
}
