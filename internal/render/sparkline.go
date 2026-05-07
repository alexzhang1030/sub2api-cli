package render

func Sparkline(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	if len(values) == 1 {
		return string(blocks[0])
	}
	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	out := make([]rune, len(values))
	if max == min {
		for i := range out {
			out[i] = blocks[0]
		}
		return string(out)
	}
	span := max - min
	for i, v := range values {
		idx := int((v - min) * int64(len(blocks)-1) / span)
		out[i] = blocks[idx]
	}
	return string(out)
}
