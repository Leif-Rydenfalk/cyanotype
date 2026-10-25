package bom

import (
	"encoding/csv"
	"fmt"
	"os"
)

func getHeader() []string {
	return []string{"Name", "Quantity"}
}

func FmtCounterAsCSV(counter map[string]float64) {
	writer := csv.NewWriter(os.Stdout)
	writer.Write(getHeader())
	for name, qty := range counter {
		line := []string{name,
			fmt.Sprintf("%.2f", qty),
		}
		writer.Write(line)
	}
	writer.Flush()
}

func FmtCounterAsList(counter map[string]float64) {
	for key := range counter {
		fmt.Println(key)
	}
}
