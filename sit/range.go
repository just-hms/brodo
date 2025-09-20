package sit

type Range struct {
	StartPoint Point
	EndPoint   Point
}

func (rng Range) Contains(pt Point) bool {
	return cmp(rng.StartPoint, pt) <= 0 && cmp(rng.EndPoint, pt) >= 0
}
