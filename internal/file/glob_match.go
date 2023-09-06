package file

func GlobMatch(pattern, name string) bool {
	px := 0
	nx := 0
	nextPx := 0
	nextNx := 0
	for px < len(pattern) || nx < len(name) {
		if px < len(pattern) {
			c := pattern[px]
			switch c {
			default:
				if nx < len(name) && name[nx] == c {
					px++
					nx++
					continue
				}
			case '?':
				if nx < len(name) {
					px++
					nx++
					continue
				}
			case '*':

				nextPx = px
				nextNx = nx + 1
				px++
				continue
			}
		}

		if 0 < nextNx && nextNx <= len(name) {
			px = nextPx
			nx = nextNx
			continue
		}
		return false
	}

	return true
}
