package storage

import "testing"

func TestResolveDataDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty uses default", in: "", want: DefaultDataDir},
		{name: "whitespace uses default", in: "  ", want: DefaultDataDir},
		{name: "current default preserved", in: DefaultDataDir, want: DefaultDataDir},
		{name: "legacy var lib lowercase remapped", in: "/var/lib/v2raye", want: DefaultDataDir},
		{name: "legacy var lib camel remapped", in: "/var/lib/v2rayE", want: DefaultDataDir},
		{name: "legacy var opt remapped", in: "/var/opt/v2rayE", want: DefaultDataDir},
		{name: "custom path preserved", in: "/srv/v2raye-data", want: "/srv/v2raye-data"},
		{name: "custom path cleaned", in: "/srv/v2raye-data/../v2raye", want: "/srv/v2raye"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveDataDir(test.in); got != test.want {
				t.Fatalf("ResolveDataDir(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}