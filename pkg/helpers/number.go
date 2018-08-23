package helpers

import "fmt"

func Percent(num float64) string {
	return fmt.Sprintf("%0.2f%%", num*100)
}
