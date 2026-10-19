package main

import "testing"

func TestChannelName(t *testing.T) {
	cases := []struct{ project, want string }{{"etl-api", "devtools.etl-api"}, {"warehouse", "devtools.warehouse"}}
	for _, tc := range cases {
		t.Run(tc.project, func(t *testing.T) {
			if got := channelName(tc.project); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
