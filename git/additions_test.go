package git_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/just-hms/brodo/git"
	"github.com/just-hms/brodo/sit"
)

func TestAdditions(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want map[string][]git.Addition
	}{
		{
			name: "single file, single addition",
			diff: `diff --git a/file.txt b/file.txt
index e69de29..4b825dc 100644
--- a/file.txt
+++ b/file.txt
@@ -0,0 +1,1 @@
+hello world
`,
			want: map[string][]git.Addition{
				"file.txt": {
					{Point: sit.Point{Row: 0, Column: 0}, Content: "hello world"},
				},
			},
		},
		{
			name: "single file, multiple additions",
			diff: `diff --git a/foo.txt b/foo.txt
index e69de29..4b825dc 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,2 +1,4 @@
 context
+added line 1
 removed
+added line 2
`,
			want: map[string][]git.Addition{
				"foo.txt": {
					{Point: sit.Point{Row: 1, Column: 0}, Content: "added line 1"},
					{Point: sit.Point{Row: 3, Column: 0}, Content: "added line 2"},
				},
			},
		},
		{
			name: "two files with additions",
			diff: `diff --git a/a.txt b/a.txt
index e69de29..4b825dc 100644
--- a/a.txt
+++ b/a.txt
@@ -0,0 +1,1 @@
+line a1
diff --git a/b.txt b/b.txt
index e69de29..4b825dc 100644
--- a/b.txt
+++ b/b.txt
@@ -0,0 +1,1 @@
+line b1
`,
			want: map[string][]git.Addition{
				"a.txt": {{Point: sit.Point{Row: 0, Column: 0}, Content: "line a1"}},
				"b.txt": {{Point: sit.Point{Row: 0, Column: 0}, Content: "line b1"}},
			},
		},
		{
			name: "deleted file has no additions",
			diff: `diff --git a/old.txt b/old.txt
deleted file mode 100644
index e69de29..0000000
--- a/old.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-old content
`,
			want: map[string][]git.Addition{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := git.Additions(tt.diff)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}
