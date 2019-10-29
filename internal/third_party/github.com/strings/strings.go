package strings

import (
	"encoding/csv"
	std_strings "strings"
)

// Origin:
//   https://github.com/spf13/viper/blob/6d33b5a963d922d182c91e8a1c88d81fd150cfd4/viper.go#L1073
//   MIT: https://github.com/spf13/viper/blob/6d33b5a963d922d182c91e8a1c88d81fd150cfd4/LICENSE
func ReadAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}
	stringReader := std_strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}
