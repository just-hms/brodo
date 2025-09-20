package git

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/just-hms/brodo/sit"
)

type Addition struct {
	sit.Point
	Content string
}

// Additions parses a unified git diff string and returns a map of filenames to
// the positions (line, column) of added lines in the new file version.
func Additions(diff string) map[string][]Addition {
	result := make(map[string][]Addition)

	var currentFile string
	var newLineNum int
	var inHunk bool

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()

		// File header line: "+++ b/path/to/file"
		if path, ok := strings.CutPrefix(line, "+++ "); ok {
			// Strip leading "a/" or "b/" if present
			if len(path) > 2 && (strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/")) {
				path = path[2:]
			}
			currentFile = path
			continue
		}

		// Hunk header: "@@ -oldStart,oldCount +newStart,newCount @@"
		if strings.HasPrefix(line, "@@ ") {
			inHunk = true
			// Extract +newStart
			parts := strings.Split(line, " ")
			for _, part := range parts {
				if numPart, ok := strings.CutPrefix(part, "+"); ok {
					if idx := strings.Index(numPart, ","); idx != -1 {
						numPart = numPart[:idx]
					}
					if n, err := parseInt(numPart); err == nil {
						newLineNum = n
					}
					break
				}
			}
			continue
		}

		if inHunk && currentFile != "" {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				// Added line
				point := Addition{
					Content: strings.TrimPrefix(line, "+"),
					Point: sit.Point{
						Row:    uint32(newLineNum - 1), // sitter.Point.Row is 0-based
						Column: 0,                      // Column unknown, default to start
					},
				}
				result[currentFile] = append(result[currentFile], point)
				newLineNum++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				// Removed line — don't increment newLineNum
			} else {
				// Context line
				newLineNum++
			}
		}
	}

	return result
}

func parseInt(s string) (int, error) {
	var n int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, &strconv.NumError{Func: "Atoi", Num: s, Err: strconv.ErrSyntax}
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}
