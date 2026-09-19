package resumes

import (
	"os"
	"path/filepath"
	"testing"
)

// setup creates a temp resume dir containing the given file names and points
// RESUME_DIR at it.
func setup(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("%PDF-1.4"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	t.Setenv(DirEnv, dir)
	return dir
}

func TestPathPrefersExactTrackName(t *testing.T) {
	setup(t, "fullstack.pdf", "Jane_Doe_Fullstack.pdf", "default.pdf")

	got, err := Path("fullstack")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := "resumes/fullstack.pdf"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPathMatchesNameTokens(t *testing.T) {
	cases := []struct {
		track string
		file  string
	}{
		{"fullstack", "Jane_Doe_Fullstack.pdf"},
		{"blockchain", "Jane_Doe_Blockchain.pdf"},
		{"fde", "Jane_Doe_Forward_Deployed.pdf"},
		{"fde", "Jane_Doe_AI_Engineer.pdf"},
		{"blockchain", "candidate-web3.pdf"},
		{"fullstack", "jane-full-stack-2026.pdf"},
		{"blockchain", "candidate_Solidity.pdf"},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			setup(t, tc.file)

			got, err := Path(tc.track)
			if err != nil {
				t.Fatalf("Path(%q): %v", tc.track, err)
			}
			if want := "resumes/" + tc.file; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

// "web" must not be found inside "web3": tokens match whole words only.
func TestPathTokenBoundaries(t *testing.T) {
	setup(t, "candidate-web3.pdf")

	got, err := Path("fullstack")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := "resumes/fullstack.pdf"; got != want {
		t.Fatalf("web3 wrongly matched fullstack: got %q, want placeholder %q", got, want)
	}
}

func TestPathPrefersStrongerSignal(t *testing.T) {
	setup(t, "ai.pdf", "ai_engineer.pdf")

	got, err := Path("fde")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := "resumes/ai_engineer.pdf"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPathIsDeterministicOnTies(t *testing.T) {
	setup(t, "b_fullstack.pdf", "a_fullstack.pdf")

	for i := 0; i < 3; i++ {
		got, err := Path("fullstack")
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := "resumes/a_fullstack.pdf"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestPathFallsBackToDefaultFile(t *testing.T) {
	setup(t, "default.pdf", "totally_unrelated.pdf")

	got, err := Path("blockchain")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := "resumes/default.pdf"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPathFallsBackToSinglePDF(t *testing.T) {
	setup(t, "My_Only_Resume.pdf")

	got, err := Path("fde")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := "resumes/My_Only_Resume.pdf"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPathReturnsPlaceholderWhenNoPDFs(t *testing.T) {
	setup(t)

	got, err := Path("fde")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := "resumes/fde.pdf"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if Exists(got) {
		t.Fatalf("placeholder %q should not exist", got)
	}
}

func TestPathRejectsUnknownTrack(t *testing.T) {
	setup(t, "fullstack.pdf")

	if _, err := Path("marketing"); err == nil {
		t.Fatal("expected an error for an unsupported track")
	}
}

func TestTrackInverse(t *testing.T) {
	cases := map[string]string{
		"resumes/Jane_Doe_Forward_Deployed.pdf": "fde",
		"resumes/fullstack.pdf":              "fullstack",
		"resumes/Jane_Doe_Blockchain.pdf":    "blockchain",
		"resumes/candidate-web3.pdf":         "blockchain",
		"":                                   "fullstack",
		"resumes/whatever.pdf":               "fullstack",
	}

	for path, want := range cases {
		if got := Track(path); got != want {
			t.Errorf("Track(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestDirPrefersEnvThenSiblings(t *testing.T) {
	t.Run("env wins", func(t *testing.T) {
		dir := setup(t, "fullstack.pdf")
		if got := Dir(); got != dir {
			t.Fatalf("Dir() = %q, want %q", got, dir)
		}
	})

	t.Run("local resumes dir", func(t *testing.T) {
		work := t.TempDir()
		if err := os.MkdirAll(filepath.Join(work, DefaultDir), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv(DirEnv, "")
		t.Chdir(work)

		if got := Dir(); got != DefaultDir {
			t.Fatalf("Dir() = %q, want %q", got, DefaultDir)
		}
	})

	t.Run("parent resumes dir", func(t *testing.T) {
		work := t.TempDir()
		if err := os.MkdirAll(filepath.Join(work, DefaultDir), 0o755); err != nil {
			t.Fatal(err)
		}
		child := filepath.Join(work, "backend")
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv(DirEnv, "")
		t.Chdir(child)

		if got := Dir(); got != filepath.Join("..", DefaultDir) {
			t.Fatalf("Dir() = %q, want %q", got, filepath.Join("..", DefaultDir))
		}
	})
}

func TestInspectReportsTracksAndGaps(t *testing.T) {
	t.Run("dedicated and fallback files", func(t *testing.T) {
		setup(t, "Jane_Doe_Fullstack.pdf", "default.pdf")

		st := Inspect()
		if st.Tracks["fullstack"] != "resumes/Jane_Doe_Fullstack.pdf" {
			t.Errorf("fullstack = %q", st.Tracks["fullstack"])
		}
		if st.Tracks["fde"] != "resumes/default.pdf" {
			t.Errorf("fde = %q, want the default file", st.Tracks["fde"])
		}
		if len(st.Missing) != 0 {
			t.Errorf("Missing = %v, want none", st.Missing)
		}
		if st.Fallback != "resumes/default.pdf" {
			t.Errorf("Fallback = %q", st.Fallback)
		}
	})

	t.Run("missing tracks are reported", func(t *testing.T) {
		setup(t, "fullstack.pdf")

		st := Inspect()
		if len(st.Missing) != 2 {
			t.Fatalf("Missing = %v, want blockchain and fde", st.Missing)
		}
	})
}

func TestAvailableListsEntries(t *testing.T) {
	setup(t, "Jane_Doe_Forward_Deployed.pdf", "default.pdf")

	entries := Available()
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].File != "Jane_Doe_Forward_Deployed.pdf" || entries[0].Track != "fde" {
		t.Errorf("first entry = %+v", entries[0])
	}
	if !entries[1].Default {
		t.Errorf("default.pdf should be flagged as the fallback: %+v", entries[1])
	}
}