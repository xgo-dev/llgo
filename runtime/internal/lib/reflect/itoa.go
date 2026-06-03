package reflect

// Local decimal formatting avoids depending on an alt-only internal helper.
func itoa(val int) string {
	if val < 0 {
		return "-" + uitoa(uint(-val))
	}
	return uitoa(uint(val))
}

func uitoa(val uint) string {
	if val == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for val >= 10 {
		q := val / 10
		buf[i] = byte('0' + val - q*10)
		i--
		val = q
	}
	buf[i] = byte('0' + val)
	return string(buf[i:])
}
