package runtime

import "strconv"

func itoa(n uint64) string {
	return strconv.FormatUint(n, 10)
}
