package scrapers

import "testing"

// Every spec must build a scraper whose Name matches the spec name. The database
// stores jobs.source and the dashboard keys paywall notes off it, so a mismatch
// silently breaks both — which is exactly how the crypto source ended up hidden
// behind the wrong name.
func TestRegistrySpecNameMatchesScraperName(t *testing.T) {
	for _, spec := range Registry {
		t.Run(spec.Name, func(t *testing.T) {
			if spec.Build == nil {
				t.Fatal("Build is nil")
			}
			if got := spec.Build(Deps{}).Name(); got != spec.Name {
				t.Fatalf("scraper Name() = %q, spec Name = %q", got, spec.Name)
			}
		})
	}
}

func TestRegistryNamesAndLabelsAreSet(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range Registry {
		if spec.Name == "" {
			t.Fatal("a spec has an empty Name")
		}
		if spec.Label == "" {
			t.Errorf("%s has an empty Label", spec.Name)
		}
		if seen[spec.Name] {
			t.Errorf("duplicate source name %q", spec.Name)
		}
		seen[spec.Name] = true
	}
}

// The crypto source must stay named "web3": that is the value already stored in
// jobs.source, and renaming it would orphan existing rows and their badges.
func TestWeb3SourceNameIsStable(t *testing.T) {
	spec, ok := Lookup("web3")
	if !ok {
		t.Fatal("web3 source is missing from the registry")
	}
	if spec.Label != "Crypto / Web3" {
		t.Errorf("Label = %q", spec.Label)
	}
	if spec.Note == "" {
		t.Error("the RemoteOK-derived crypto feed should carry a paywall note")
	}
}

func TestReadyRequiresEnvKeys(t *testing.T) {
	spec := Spec{Name: "keyed", EnvKeys: []string{"TEST_SOURCE_API_KEY"}}
	t.Setenv("TEST_SOURCE_API_KEY", "")
	if ok, reason := spec.Ready(); ok || reason == "" {
		t.Fatalf("Ready() = %v, %q; want not ready with a reason", ok, reason)
	}
	t.Setenv("TEST_SOURCE_API_KEY", "secret")
	if ok, reason := spec.Ready(); !ok {
		t.Fatalf("Ready() = false, %q; want ready once the key is set", reason)
	}
}

func TestReadyRequiresOptIn(t *testing.T) {
	spec := Spec{Name: "gated", OptInEnv: "TEST_GATED_ENABLED"}
	if ok, reason := spec.Ready(); ok || reason == "" {
		t.Fatalf("off by default expected, got %v %q", ok, reason)
	}
	t.Setenv("TEST_GATED_ENABLED", "true")
	if ok, _ := spec.Ready(); !ok {
		t.Fatal("expected ready when the opt-in flag is true")
	}
}

func TestSupportsCountry(t *testing.T) {
	global := Spec{Name: "global"}
	if !global.SupportsCountry("ae") {
		t.Error("a source with no Countries should support every country")
	}
	scoped := Spec{Name: "scoped", Countries: []string{"au", "nz"}}
	if !scoped.SupportsCountry("AU") {
		t.Error("country matching should be case-insensitive")
	}
	if scoped.SupportsCountry("ae") {
		t.Error("UAE is not covered by an AU/NZ source")
	}
}

func TestEnabledRespectsSettings(t *testing.T) {
	ready := 0
	for _, spec := range Registry {
		if ok, _ := spec.Ready(); ok {
			ready++
		}
	}
	if got := len(Enabled(nil)); got != ready {
		t.Fatalf("Enabled(nil) = %d sources, want %d ready sources", got, ready)
	}
	if ready == 0 {
		t.Skip("no ready sources to disable")
	}
	// Switch one source off explicitly and confirm it is dropped.
	name := ""
	for _, spec := range Registry {
		if ok, _ := spec.Ready(); ok {
			name = spec.Name
			break
		}
	}
	got := Enabled(map[string]bool{name: false})
	if len(got) != ready-1 {
		t.Fatalf("Enabled() = %d sources, want %d", len(got), ready-1)
	}
	for _, spec := range got {
		if spec.Name == name {
			t.Fatalf("%s should have been switched off", name)
		}
	}
}

func TestInfoShapeIsStable(t *testing.T) {
	info := Info(nil)
	if len(info) != len(Registry) {
		t.Fatalf("Info() returned %d entries, want %d", len(info), len(Registry))
	}
	for i := 1; i < len(info); i++ {
		if info[i-1].Label > info[i].Label {
			t.Fatalf("Info() is not sorted by label: %q before %q", info[i-1].Label, info[i].Label)
		}
	}
	for _, s := range info {
		// The dashboard maps over these; nil would serialise as null and crash it.
		if s.Countries == nil || s.EnvKeys == nil {
			t.Errorf("%s has nil slices (countries/keys)", s.Name)
		}
		if s.Name == "" || s.Label == "" {
			t.Errorf("incomplete source info: %+v", s)
		}
		if s.Ready != (s.Reason == "") {
			t.Errorf("%s: Ready=%v but Reason=%q", s.Name, s.Ready, s.Reason)
		}
	}
}
