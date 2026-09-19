package textutil

import "testing"

func TestHTMLToTextParagraphsAndEntities(t *testing.T) {
	in := "<p>About Mesh</p><p>Join us!</p><h2>Overview</h2>" +
		"<p>We are looking&#8230; connect&nbsp; CEXs</p>" +
		"<ul><li>a</li><li>b</li></ul>"
	want := "About Mesh\n\nJoin us!\n\nOverview\n\nWe are looking… connect CEXs\n\na\n\nb"
	if got := HTMLToText(in); got != want {
		t.Fatalf("HTMLToText() =\n%q\nwant\n%q", got, want)
	}
}

func TestHTMLToTextBrAndInlineTags(t *testing.T) {
	in := "line one<br>line two<br/>line three" +
		"<p><strong>Bold</strong> and <em>italic</em> &amp; more</p>"
	want := "line one\nline two\nline three\n\nBold and italic & more"
	if got := HTMLToText(in); got != want {
		t.Fatalf("HTMLToText() =\n%q\nwant\n%q", got, want)
	}
}

func TestHTMLToTextDropsScriptAndStyle(t *testing.T) {
	in := "<style>.x{color:red}</style><p>Hello</p>" +
		"<script>alert('x')</script><p>World</p>"
	want := "Hello\n\nWorld"
	if got := HTMLToText(in); got != want {
		t.Fatalf("HTMLToText() =\n%q\nwant\n%q", got, want)
	}
}

func TestHTMLToTextPlainPassthrough(t *testing.T) {
	in := "Plain text stays untouched.\n\nSecond paragraph."
	if got := HTMLToText(in); got != in {
		t.Fatalf("HTMLToText() = %q, want unchanged %q", got, in)
	}
}

func TestHTMLToTextIdempotent(t *testing.T) {
	in := "<p>a&nbsp;&amp; b</p><div>c<br>d</div>"
	once := HTMLToText(in)
	if twice := HTMLToText(once); once != twice {
		t.Fatalf("not idempotent: %q vs %q", once, twice)
	}
}

func TestHTMLToTextEmpty(t *testing.T) {
	if got := HTMLToText(""); got != "" {
		t.Fatalf("HTMLToText(\"\") = %q, want empty", got)
	}
}
