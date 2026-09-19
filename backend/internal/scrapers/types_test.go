package scrapers

import (
	"testing"
	"time"
)

func TestParseISOTime(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantNil bool
		want    time.Time
	}{
		{
			name: "rfc3339 with offset is normalized to UTC",
			in:   "2026-09-17T16:40:14-04:00",
			want: time.Date(2026, 9, 17, 20, 40, 14, 0, time.UTC),
		},
		{
			name: "fractional seconds",
			in:   "2025-10-30T16:47:27.463+00:00",
			want: time.Date(2025, 10, 30, 16, 47, 27, 463000000, time.UTC),
		},
		{
			name: "date only",
			in:   "2026-08-06",
			want: time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC),
		},
		{name: "empty", in: "", wantNil: true},
		{name: "whitespace", in: "   ", wantNil: true},
		{name: "garbage", in: "not-a-date", wantNil: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseISOTime(c.in)
			if c.wantNil {
				if got != nil {
					t.Fatalf("parseISOTime(%q) = %v, want nil", c.in, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseISOTime(%q) = nil, want %v", c.in, c.want)
			}
			if !got.Equal(c.want) {
				t.Fatalf("parseISOTime(%q) = %v, want %v", c.in, *got, c.want)
			}
		})
	}
}

func TestParseEpochTime(t *testing.T) {
	seconds := int64(1789585202)
	millis := int64(1782214185805)

	cases := []struct {
		name    string
		in      string
		wantNil bool
		want    time.Time
	}{
		{name: "seconds", in: "1789585202", want: time.Unix(seconds, 0).UTC()},
		{name: "milliseconds are scaled down", in: "1782214185805", want: time.Unix(millis/1000, 0).UTC()},
		{name: "empty", in: "", wantNil: true},
		{name: "zero", in: "0", wantNil: true},
		{name: "negative", in: "-1", wantNil: true},
		{name: "garbage", in: "abc", wantNil: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseEpochTime(c.in)
			if c.wantNil {
				if got != nil {
					t.Fatalf("parseEpochTime(%q) = %v, want nil", c.in, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseEpochTime(%q) = nil, want %v", c.in, c.want)
			}
			if !got.Equal(c.want) {
				t.Fatalf("parseEpochTime(%q) = %v, want %v", c.in, *got, c.want)
			}
		})
	}
}

func TestParseAnyEpoch(t *testing.T) {
	want := time.Unix(1789585202, 0).UTC()

	if got := parseAnyEpoch(float64(1789585202)); got == nil || !got.Equal(want) {
		t.Fatalf("parseAnyEpoch(float64) = %v, want %v", got, want)
	}
	if got := parseAnyEpoch(int64(1789585202)); got == nil || !got.Equal(want) {
		t.Fatalf("parseAnyEpoch(int64) = %v, want %v", got, want)
	}
	if got := parseAnyEpoch("1789585202"); got == nil || !got.Equal(want) {
		t.Fatalf("parseAnyEpoch(string) = %v, want %v", got, want)
	}
	if got := parseAnyEpoch(nil); got != nil {
		t.Fatalf("parseAnyEpoch(nil) = %v, want nil", got)
	}
}

func TestFirstTime(t *testing.T) {
	first := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	second := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)

	if got := firstTime(nil, &first, &second); got == nil || !got.Equal(first) {
		t.Fatalf("firstTime() = %v, want %v", got, first)
	}
	if got := firstTime(nil, nil); got != nil {
		t.Fatalf("firstTime(all nil) = %v, want nil", got)
	}
}
