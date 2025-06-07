package text

import "fmt"

func DecorateIntWithCommas(number int) string {
	str := fmt.Sprintf("%d", number)
	for i := len(str) - 3; i > 0; i -= 3 {
		str = str[:i] + "," + str[i:]
	}
	return str
}

func DecorateFloatWithCommas(number float64) string {
	str := fmt.Sprintf("%.2f", number)
	parts := make([]string, 0)
	if idx := len(str) - 3; idx > 0 {
		parts = append(parts, str[:idx])
		parts = append(parts, str[idx:])
	} else {
		parts = append(parts, str)
	}
	for i := len(parts[0]) - 3; i > 0; i -= 3 {
		parts[0] = parts[0][:i] + "," + parts[0][i:]
	}
	return parts[0] + parts[1]
}
