package text

import "fmt"

func DecorateNumberWithCommas(number int) string {
	str := fmt.Sprintf("%d", number)
	for i := len(str) - 3; i > 0; i -= 3 {
		str = str[:i] + "," + str[i:]
	}
	return str
}
