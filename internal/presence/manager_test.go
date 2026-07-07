package presence

import "testing"

func TestFormatCompact(t *testing.T) {
	cases := map[int]string{
		0:       "0",
		2:       "2",
		869:     "869",
		999:     "999",
		1000:    "1k",
		13660:   "13.66k",
		2530000: "2.53M",
	}
	for n, want := range cases {
		if got := formatCompact(n); got != want {
			t.Fatalf("formatCompact(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestAssetResolverResolve(t *testing.T) {
	r := &assetResolver{
		byName: map[string]string{
			"iran_logo":  "1523290913651294248",
			"stats_icon": "1523290912623558808",
		},
	}

	cases := map[string]string{
		"iran_logo":                              "1523290913651294248",
		"IRAN_LOGO":                              "1523290913651294248",
		"1523290913651294248":                    "1523290913651294248",
		"https://cdn.discordapp.com/example.png": "https://cdn.discordapp.com/example.png",
		"unknown_key":                            "unknown_key",
		"":                                       "",
	}
	for input, want := range cases {
		if got := r.resolve(input); got != want {
			t.Fatalf("resolve(%q) = %q, want %q", input, got, want)
		}
	}
}
