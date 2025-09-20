package sit

import (
	"bufio"
	"os"
)

// TODO: for now this is a mock implementation, treesitter will be included in next versions
func Comments(file string) ([]*Range, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []*Range
	scanner := bufio.NewScanner(f)
	lineNum := uint32(0)

	for ; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		results = append(results, &Range{
			StartPoint: Point{Row: lineNum, Column: 0},
			EndPoint:   Point{Row: lineNum, Column: uint32(len(line))},
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil

}
