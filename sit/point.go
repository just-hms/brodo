package sit

type Point struct {
	Row    uint32
	Column uint32
}

func cmp(a, b Point) int {
	if a.Row < b.Row || (a.Row == b.Row && a.Column < b.Column) {
		return -1
	} else if a.Row > b.Row || (a.Row == b.Row && a.Column > b.Column) {
		return 1
	}
	return 0
}
