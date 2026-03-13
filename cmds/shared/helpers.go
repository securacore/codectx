package shared

// BoolCount returns the number of true values among the given bools.
// Used to detect conflicting mutually-exclusive CLI flags.
func BoolCount(vals ...bool) int {
	n := 0
	for _, v := range vals {
		if v {
			n++
		}
	}
	return n
}
