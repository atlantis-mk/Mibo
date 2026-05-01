package listener

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestLocalEventKindNormalizesFileSystemEvents(t *testing.T) {
	cases := []struct {
		name string
		op   fsnotify.Op
		want string
	}{
		{name: "create", op: fsnotify.Create, want: "create"},
		{name: "write", op: fsnotify.Write, want: "update"},
		{name: "chmod", op: fsnotify.Chmod, want: "update"},
		{name: "remove", op: fsnotify.Remove, want: "delete"},
		{name: "rename", op: fsnotify.Rename, want: "delete"},
		{name: "create wins over write only after delete checked", op: fsnotify.Create | fsnotify.Write, want: "create"},
		{name: "unknown", op: 0, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := localEventKind(fsnotify.Event{Name: "/media/file.mkv", Op: tc.op})
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
