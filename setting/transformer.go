package setting

import (
	"os"
	"strconv"
)

var TransformerEnabled = true

func init() {
	if value := os.Getenv("TRANSFORMER_ENABLED"); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err == nil {
			TransformerEnabled = enabled
		}
	}
}
