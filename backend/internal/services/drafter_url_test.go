package services

import "testing"

func TestLLMBaseURL(t *testing.T) {
	cases := []struct {
		name, base, alt, want string
	}{
		{"default when unset", "", "", "https://api.openai.com/v1"},
		{"explicit base", "https://openrouter.ai/api/v1", "", "https://openrouter.ai/api/v1"},
		{"trailing slash trimmed", "https://api.groq.com/openai/v1/", "", "https://api.groq.com/openai/v1"},
		{"OPENAI_BASE_URL fallback", "", "http://localhost:11434/v1", "http://localhost:11434/v1"},
		{"primary wins over fallback", "https://a.example/v1", "https://b.example/v1", "https://a.example/v1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("OPENAI_API_BASE_URL", c.base)
			t.Setenv("OPENAI_BASE_URL", c.alt)
			if got := LLMBaseURL(); got != c.want {
				t.Fatalf("LLMBaseURL() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestChatCompletionsURL(t *testing.T) {
	cases := []struct {
		name, base, want string
	}{
		{"base gets path appended", "https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"full endpoint kept as-is", "https://api.cline.bot/api/v1/chat/completions", "https://api.cline.bot/api/v1/chat/completions"},
		{"trailing slash handled", "https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"default base", "", "https://api.openai.com/v1/chat/completions"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("OPENAI_API_BASE_URL", c.base)
			t.Setenv("OPENAI_BASE_URL", "")
			if got := chatCompletionsURL(); got != c.want {
				t.Fatalf("chatCompletionsURL() = %q, want %q", got, c.want)
			}
		})
	}
}
