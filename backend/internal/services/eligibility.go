package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/mueedx/job-bot/backend/internal/models"
)

// Verdict statuses stored on jobs.eligibility.
const (
	EligibilityPass    = "pass"
	EligibilityVeto    = "veto"
	EligibilityUnknown = "unknown"
)

// Rule identifiers stored on jobs.eligibility_rule: which rule decided the
// verdict. Shown verbatim by the dashboard, so keep them stable.
const (
	RuleDisabled              = "disabled"
	RuleWorldwideRemote       = "worldwide_remote"
	RuleSponsoredRelocation   = "sponsored_relocation"
	RuleAuthorizedCountry     = "authorized_country_remote"
	RuleCountryBoundedRemote  = "country_bounded_remote"
	RuleLocalWorkPermit       = "local_work_permit"
	RuleNoSignal              = "no_signal"
	RuleUnknownVetoByOperator = "unknown_vetoed"
	// StatusRejected is the pipeline status a vetoed job is moved to (Archived).
	StatusRejected = "rejected"
)

// EligibilityRules is the operator-owned eligibility policy. Every field is
// optional in settings.yaml: a missing key keeps the default, so a hand-edited
// file can never silently disable a rule. Pointers distinguish "unset" from
// "explicitly false".
//
// Phrase lists are matched case-insensitively on word boundaries against the
// posting's title, location and description. `{country}` in a phrase stands for
// any recognised country name or unambiguous country code. An explicitly empty
// list (`worldwide_phrases: []`) disables that signal; an omitted list keeps the
// defaults.
type EligibilityRules struct {
	Enabled *bool `json:"enabled" yaml:"enabled"`

	// WorldwideRemoteOK passes worldwide / global remote / hire-anywhere roles.
	WorldwideRemoteOK *bool `json:"worldwide_remote_ok" yaml:"worldwide_remote_ok"`
	// SponsoredRelocationOK passes on-site or hybrid roles that offer visa
	// sponsorship and relocation assistance, and flags the draft to highlight
	// relocation readiness.
	SponsoredRelocationOK *bool `json:"sponsored_relocation_ok" yaml:"sponsored_relocation_ok"`
	// VetoCountryBoundedRemote hard-rejects remote roles bounded to a country
	// the operator is not work-authorized in.
	VetoCountryBoundedRemote *bool `json:"veto_country_bounded_remote" yaml:"veto_country_bounded_remote"`
	// VetoLocalWorkPermit hard-rejects roles that require existing local
	// citizenship, residency, or a work permit.
	VetoLocalWorkPermit *bool `json:"veto_local_work_permit" yaml:"veto_local_work_permit"`
	// HonorWorkAuthorizedCountries exempts country-bounded roles in countries
	// listed in WorkAuthorizedCountries from that veto.
	HonorWorkAuthorizedCountries *bool `json:"honor_work_authorized_countries" yaml:"honor_work_authorized_countries"`
	// VetoUnknown rejects postings with no eligibility signal at all. Off by
	// default: a veto must be evidence, not a guess.
	VetoUnknown *bool `json:"veto_unknown" yaml:"veto_unknown"`

	// WorkAuthorizedCountries are the lowercase ISO-3166 alpha-2 codes where the
	// operator already holds citizenship or a work permit.
	WorkAuthorizedCountries []string `json:"work_authorized_countries" yaml:"work_authorized_countries"`

	WorldwidePhrases        []string `json:"worldwide_phrases" yaml:"worldwide_phrases"`
	SponsorshipPhrases      []string `json:"sponsorship_phrases" yaml:"sponsorship_phrases"`
	RelocationPhrases       []string `json:"relocation_phrases" yaml:"relocation_phrases"`
	CountryBoundedPhrases   []string `json:"country_bounded_phrases" yaml:"country_bounded_phrases"`
	WorkPermitPhrases       []string `json:"work_permit_phrases" yaml:"work_permit_phrases"`
	WorkPermitCountryPhrase []string `json:"work_permit_country_phrases" yaml:"work_permit_country_phrases"`

	// RelocationHint is the extra instruction given to the drafter when a role
	// passed on sponsorship + relocation.
	RelocationHint string `json:"relocation_hint" yaml:"relocation_hint"`
}

// EligibilityVerdict is the outcome of evaluating one posting.
type EligibilityVerdict struct {
	Status               string   `json:"status"` // pass | veto | unknown
	Rule                 string   `json:"rule"`
	Reason               string   `json:"reason"`
	Signals              []string `json:"signals"`
	Countries            []string `json:"countries"`
	HighlightsRelocation bool     `json:"highlights_relocation"`
}

// DefaultEligibilityRules is the stated policy, ready to use before the operator
// configures anything: worldwide and sponsored-relocation roles pass, country
// bounded remote and existing-work-rights roles are vetoed. WorkAuthorizedCountries
// is intentionally empty — the operator states their own work rights.
func DefaultEligibilityRules() *EligibilityRules {
	return &EligibilityRules{
		Enabled:                      boolPtr(true),
		WorldwideRemoteOK:            boolPtr(true),
		SponsoredRelocationOK:        boolPtr(true),
		VetoCountryBoundedRemote:     boolPtr(true),
		VetoLocalWorkPermit:          boolPtr(true),
		HonorWorkAuthorizedCountries: boolPtr(true),
		VetoUnknown:                  boolPtr(false),
		WorkAuthorizedCountries:      []string{},

		WorldwidePhrases: []string{
			"worldwide", "world wide", "work from anywhere", "work anywhere",
			"anywhere in the world", "remote worldwide", "globally remote",
			"global remote", "remote anywhere", "hire anywhere", "hiring anywhere",
			"any location", "any timezone", "any time zone", "location independent",
			"location agnostic", "no location restriction", "distributed team",
			"globally distributed", "work from any country", "employer of record",
			"global employment", "global payroll", "international contractor",
			"independent contractor", "contractor agreement", "deel", "remote.com",
			"oyster", "remote first",
		},
		SponsorshipPhrases: []string{
			"visa sponsorship", "sponsorship available", "sponsorship provided",
			"sponsorship offered", "we sponsor", "we will sponsor", "will sponsor",
			"can sponsor", "happy to sponsor", "sponsor a visa", "visa support",
			"immigration support", "work permit support", "relocation and visa",
			"visa and relocation", "tier 2 sponsorship", "skilled worker visa",
			"h1b", "h 1b",
		},
		RelocationPhrases: []string{
			"relocation assistance", "relocation support", "relocation package",
			"relocation provided", "relocation allowance", "relocation bonus",
			"relocation offered", "relocation benefits", "relocation included",
			"relocation covered", "relocation help", "assistance with relocation",
			"help you relocate", "help with your move", "moving costs covered",
			"we will relocate you",
		},
		CountryBoundedPhrases: []string{
			"remote {country} only", "remote in {country} only", "remote within {country}",
			"remote {country} based", "remote position in {country}", "remote role in {country}",
			"remote job in {country}", "anywhere in {country}", "anywhere within {country}",
			"{country} remote only", "{country} only remote", "{country} based remote",
			"must reside in {country}", "must be resident in {country}",
			"must be a resident of {country}", "must be based in {country}",
			"must be located in {country}", "must live in {country}",
			"you must live in {country}", "based in {country} only",
			"candidates based in {country} only", "only candidates based in {country}",
			"only candidates located in {country}", "only considering candidates based in {country}",
			"only considering candidates located in {country}", "candidates must be based in {country}",
			"candidates must reside in {country}", "{country} residents only",
			"residents of {country} only", "open only to candidates in {country}",
			"open only to {country} residents", "work from {country} only",
			"only open to {country} based candidates", "position is only open to {country}",
		},
		WorkPermitPhrases: []string{
			"no visa sponsorship", "visa sponsorship is not available",
			"visa sponsorship not available", "sponsorship is not available",
			"no sponsorship available", "no sponsorship", "without sponsorship",
			"cannot sponsor", "can not sponsor", "we do not sponsor", "does not sponsor",
			"not able to sponsor", "unable to sponsor", "no visa support",
			"must have the right to work", "must have the legal right to work",
			"must have existing right to work", "must already have the right to work",
			"must be legally authorized to work", "must be authorized to work",
			"must already be authorized to work", "existing work permit",
			"valid work permit", "work permit required", "citizens only",
			"citizenship required", "must be a citizen", "local candidates only",
			"only local candidates", "security clearance", "active clearance",
			"must be a permanent resident", "permanent residency required", "green card",
		},
		WorkPermitCountryPhrase: []string{
			"authorized to work in {country}", "must be authorized to work in {country}",
			"eligible to work in {country}", "right to work in {country}",
			"work permit in {country}", "work authorization in {country}",
			"{country} work permit", "citizen of {country}", "{country} citizen",
			"permanent resident of {country}", "{country} permanent resident",
		},
		RelocationHint: "This role offers visa sponsorship or relocation support. In one short " +
			"sentence, state the candidate's relocation readiness and work-authorization facts " +
			"using only the work_authorization and relocation facts provided. Do not invent anything.",
	}
}

func boolPtr(b bool) *bool { return &b }

// WithDefaults fills unset fields with the defaults and normalises the lists.
// A nil list takes the default; an explicitly empty list stays empty.
func (r *EligibilityRules) WithDefaults() *EligibilityRules {
	out := DefaultEligibilityRules()
	out.Merge(r)
	out.Normalize()
	return out
}

// Merge overlays set fields from over onto r, so a partial hand-edited block in
// settings.yaml keeps the defaults it did not mention.
func (r *EligibilityRules) Merge(over *EligibilityRules) {
	if over == nil {
		return
	}
	apply := func(dst **bool, src *bool) {
		if src != nil {
			v := *src
			*dst = &v
		}
	}
	apply(&r.Enabled, over.Enabled)
	apply(&r.WorldwideRemoteOK, over.WorldwideRemoteOK)
	apply(&r.SponsoredRelocationOK, over.SponsoredRelocationOK)
	apply(&r.VetoCountryBoundedRemote, over.VetoCountryBoundedRemote)
	apply(&r.VetoLocalWorkPermit, over.VetoLocalWorkPermit)
	apply(&r.HonorWorkAuthorizedCountries, over.HonorWorkAuthorizedCountries)
	apply(&r.VetoUnknown, over.VetoUnknown)

	for _, f := range []struct {
		dst *[]string
		src []string
	}{
		{&r.WorkAuthorizedCountries, over.WorkAuthorizedCountries},
		{&r.WorldwidePhrases, over.WorldwidePhrases},
		{&r.SponsorshipPhrases, over.SponsorshipPhrases},
		{&r.RelocationPhrases, over.RelocationPhrases},
		{&r.CountryBoundedPhrases, over.CountryBoundedPhrases},
		{&r.WorkPermitPhrases, over.WorkPermitPhrases},
		{&r.WorkPermitCountryPhrase, over.WorkPermitCountryPhrase},
	} {
		if f.src != nil {
			*f.dst = append([]string(nil), f.src...)
		}
	}
	if strings.TrimSpace(over.RelocationHint) != "" {
		r.RelocationHint = over.RelocationHint
	}
}

// flagOr returns a copy of the flag, or the default when it was unset. Keeps
// Normalize idempotent and makes a hand-built rules struct behave like the
// loaded one (nil means "use the default", not "false").
func flagOr(p *bool, def bool) *bool {
	if p != nil {
		v := *p
		return &v
	}
	return &def
}

// Normalize lowercases and trims every list, removing duplicates and keeping the
// order the operator wrote, so hand-edited YAML and the API agree on matching.
func (r *EligibilityRules) Normalize() *EligibilityRules {
	if r == nil {
		return r
	}
	r.Enabled = flagOr(r.Enabled, true)
	r.WorldwideRemoteOK = flagOr(r.WorldwideRemoteOK, true)
	r.SponsoredRelocationOK = flagOr(r.SponsoredRelocationOK, true)
	r.VetoCountryBoundedRemote = flagOr(r.VetoCountryBoundedRemote, true)
	r.VetoLocalWorkPermit = flagOr(r.VetoLocalWorkPermit, true)
	r.HonorWorkAuthorizedCountries = flagOr(r.HonorWorkAuthorizedCountries, true)
	r.VetoUnknown = flagOr(r.VetoUnknown, false)

	r.WorkAuthorizedCountries = normalizeList(r.WorkAuthorizedCountries)
	r.WorldwidePhrases = normalizeList(r.WorldwidePhrases)
	r.SponsorshipPhrases = normalizeList(r.SponsorshipPhrases)
	r.RelocationPhrases = normalizeList(r.RelocationPhrases)
	r.CountryBoundedPhrases = normalizeList(r.CountryBoundedPhrases)
	r.WorkPermitPhrases = normalizeList(r.WorkPermitPhrases)
	r.WorkPermitCountryPhrase = normalizeList(r.WorkPermitCountryPhrase)
	r.RelocationHint = strings.Join(strings.Fields(strings.TrimSpace(r.RelocationHint)), " ")
	return r
}

// Validate normalises the rules and reports unusable configuration (a bad
// country code, or a phrase list blanked down to nothing meaningful) instead of
// letting it silently match nothing.
func (r *EligibilityRules) Validate() error {
	if r == nil {
		return nil
	}
	r.Normalize()
	for _, c := range r.WorkAuthorizedCountries {
		if len(c) != 2 || !isLowerAlpha(c) {
			return fmt.Errorf("work_authorized_countries entry %q is not a 2-letter country code (e.g. ie, gb, pk)", c)
		}
	}
	if !on(r.Enabled) {
		return nil
	}
	checks := []struct {
		name    string
		enabled bool
		phrases []string
	}{
		{"worldwide_phrases", on(r.WorldwideRemoteOK), r.WorldwidePhrases},
		{"sponsorship_phrases", on(r.SponsoredRelocationOK), r.SponsorshipPhrases},
		{"relocation_phrases", on(r.SponsoredRelocationOK), r.RelocationPhrases},
		{"country_bounded_phrases", on(r.VetoCountryBoundedRemote), r.CountryBoundedPhrases},
		{"work_permit_phrases", on(r.VetoLocalWorkPermit), r.WorkPermitPhrases},
		{"work_permit_country_phrases", on(r.VetoLocalWorkPermit), r.WorkPermitCountryPhrase},
	}
	for _, c := range checks {
		if !c.enabled {
			continue
		}
		for _, p := range c.phrases {
			if len([]rune(p)) > 120 {
				return fmt.Errorf("%s: phrase %q is longer than 120 characters", c.name, truncateMsg(p, 40))
			}
			if !strings.ContainsAny(p, "abcdefghijklmnopqrstuvwxyz") {
				return fmt.Errorf("%s: phrase %q has no letters", c.name, p)
			}
		}
	}
	return nil
}

// normalizeList lowercases, trims, collapses whitespace, removes dashes so
// "Remote — US only" and "remote - us only" match one phrase, and de-duplicates.
func normalizeList(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, raw := range in {
		v := normalizeText(raw)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// normalizeText lowercases a posting or phrase and flattens punctuation that
// boards use interchangeably: dashes, slashes, brackets and whitespace runs.
func normalizeText(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"\u2014", " ", "\u2013", " ", "-", " ", "/", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", ",", " ", ".", " ", "\n", " ", "\r", " ", "\t", " ",
	)
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// countryAliases maps a normalised country name or unambiguous code to its
// lowercase ISO-3166 alpha-2 code. Codes that are also common English words
// (in, it, is, no, at, id, my, me) are deliberately absent so they can never be
// mistaken for a country; a literal phrase in settings.yaml covers those rare
// cases. "us"/"uk" are kept because boards write bounded roles that way.
var countryAliases = map[string]string{
	"united states": "us", "united states of america": "us", "usa": "us",
	"u s": "us", "u s a": "us", "us": "us", "america": "us",
	"united kingdom": "gb", "uk": "gb", "u k": "gb", "great britain": "gb",
	"britain": "gb", "england": "gb", "scotland": "gb", "wales": "gb",
	"northern ireland": "gb",
	"ireland":          "ie", "republic of ireland": "ie",
	"canada": "ca", "australia": "au", "new zealand": "nz", "nz": "nz",
	"germany": "de", "deutschland": "de", "france": "fr", "netherlands": "nl",
	"the netherlands": "nl", "holland": "nl", "belgium": "be", "spain": "es",
	"portugal": "pt", "italy": "it", "switzerland": "ch", "austria": "at",
	"sweden": "se", "norway": "no", "denmark": "dk", "finland": "fi",
	"iceland": "is", "poland": "pl", "czech republic": "cz", "czechia": "cz",
	"slovakia": "sk", "hungary": "hu", "romania": "ro", "bulgaria": "bg",
	"greece": "gr", "croatia": "hr", "serbia": "rs", "slovenia": "si",
	"estonia": "ee", "latvia": "lv", "lithuania": "lt", "ukraine": "ua",
	"turkey": "tr", "turkiye": "tr", "russia": "ru", "israel": "il",
	"united arab emirates": "ae", "uae": "ae", "u a e": "ae", "dubai": "ae",
	"abu dhabi": "ae", "saudi arabia": "sa", "ksa": "sa", "qatar": "qa",
	"kuwait": "kw", "bahrain": "bh", "oman": "om", "jordan": "jo", "egypt": "eg",
	"morocco": "ma", "nigeria": "ng", "kenya": "ke", "south africa": "za",
	"ghana": "gh", "pakistan": "pk", "india": "in", "bangladesh": "bd",
	"sri lanka": "lk", "nepal": "np", "china": "cn", "hong kong": "hk",
	"taiwan": "tw", "japan": "jp", "south korea": "kr", "korea": "kr",
	"singapore": "sg", "malaysia": "my", "indonesia": "id", "philippines": "ph",
	"thailand": "th", "vietnam": "vn", "brazil": "br", "argentina": "ar",
	"chile": "cl", "colombia": "co", "peru": "pe", "mexico": "mx",
	"costa rica": "cr", "uruguay": "uy",
}

var (
	countryAltOnce sync.Once
	countryAltRe   string
	countryAltNorm = map[string]string{}
)

// countryAlternation returns the regex alternation of every known country alias,
// longest first so "united states" wins over "us".
func countryAlternation() string {
	countryAltOnce.Do(func() {
		aliases := make([]string, 0, len(countryAliases))
		for alias, code := range countryAliases {
			norm := normalizeText(alias)
			countryAltNorm[norm] = code
			aliases = append(aliases, norm)
		}
		sort.Slice(aliases, func(i, j int) bool {
			if len(aliases[i]) != len(aliases[j]) {
				return len(aliases[i]) > len(aliases[j])
			}
			return aliases[i] < aliases[j]
		})
		for i, a := range aliases {
			if i > 0 {
				countryAltRe += "|"
			}
			countryAltRe += regexp.QuoteMeta(a)
		}
		countryAltRe = "(?:" + countryAltRe + ")"
	})
	return countryAltRe
}

// countryCodeFor resolves a matched alias back to its ISO code.
func countryCodeFor(alias string) string {
	countryAlternation()
	return countryAltNorm[normalizeText(alias)]
}

var regexCache sync.Map // pattern -> *regexp.Regexp

// phraseRegex compiles a phrase (or a `{country}` template) into a
// word-boundary regex. Patterns are cached: an ingest run compiles each phrase
// once no matter how many postings it evaluates.
func phraseRegex(phrase string) *regexp.Regexp {
	if cached, ok := regexCache.Load(phrase); ok {
		return cached.(*regexp.Regexp)
	}
	parts := strings.Split(phrase, "{country}")
	var b strings.Builder
	b.WriteString(`\b`)
	for i, p := range parts {
		if i > 0 {
			b.WriteString("(" + countryAlternation() + ")")
		}
		b.WriteString(regexp.QuoteMeta(p))
	}
	b.WriteString(`\b`)
	compiled, err := regexp.Compile(b.String())
	if err != nil {
		compiled = regexp.MustCompile(regexp.QuoteMeta(phrase))
	}
	regexCache.Store(phrase, compiled)
	return compiled
}

// negativeMarkers invalidate a pass signal that is really a refusal, so
// "no relocation support" can never count as relocation support.
var negativeMarkers = []string{"no", "not", "without", "cannot", "can", "unable", "unfortunately", "excluding"}

const negativeWindow = 30

func hasNegativePrefix(blob string, at int) bool {
	start := at - negativeWindow
	if start < 0 {
		start = 0
	}
	window := " " + blob[start:at] + " "
	for _, marker := range negativeMarkers {
		if strings.Contains(window, " "+marker+" ") {
			return true
		}
	}
	return false
}

// matcher evaluates phrase lists against one normalised posting blob.
type matcher struct {
	blob string
}

// match returns the phrases found in the blob. guardNegatives is used for pass
// signals; veto phrases are matched as written.
func (m matcher) match(phrases []string, guardNegatives bool) []string {
	var hits []string
	for _, p := range phrases {
		if p == "" {
			continue
		}
		re := phraseRegex(p)
		matched := false
		for _, loc := range re.FindAllStringIndex(m.blob, -1) {
			if guardNegatives && hasNegativePrefix(m.blob, loc[0]) {
				continue
			}
			matched = true
			break
		}
		if matched {
			hits = append(hits, p)
		}
	}
	return hits
}

// countryMatch is a country found in a posting, with the alias as written.
type countryMatch struct {
	Code  string
	Alias string
}

// matchCountries returns the `{country}` phrases that matched and the countries
// they name.
func (m matcher) matchCountries(phrases []string) ([]string, []countryMatch) {
	var hits []string
	var textHits []string
	var found []countryMatch
	seen := map[string]bool{}
	seenText := map[string]bool{}
	for _, p := range phrases {
		if p == "" || !strings.Contains(p, "{country}") {
			continue
		}
		hit := false
		for _, match := range phraseRegex(p).FindAllStringSubmatch(m.blob, -1) {
			if len(match) < 2 {
				continue
			}
			code := countryCodeFor(match[1])
			if code == "" {
				continue
			}
			hit = true
			// Report the text as it appeared in the posting, not the template:
			// the verdict reason is read by a human checking the posting.
			if !seen[code] {
				seen[code] = true
				found = append(found, countryMatch{Code: code, Alias: normalizeText(match[1])})
			}
			if !seenText[match[0]] {
				seenText[match[0]] = true
				textHits = append(textHits, match[0])
			}
		}
		if hit {
			hits = append(hits, p)
		}
	}
	return unique(append(textHits, hits...)), found
}

// literalPhrases are configured phrases without a `{country}` placeholder. They
// are matched literally and always veto when they hit, which is the escape hatch
// for region wording ("remote eu only") the country table cannot resolve.
func literalPhrases(phrases []string) []string {
	var out []string
	for _, p := range phrases {
		if p != "" && !strings.Contains(p, "{country}") {
			out = append(out, p)
		}
	}
	return out
}

// detectCountries lists every country named in a normalised blob, in order.
func detectCountries(blob string) []countryMatch {
	re := regexp.MustCompile(`\b(` + countryAlternation() + `)\b`)
	var found []countryMatch
	seen := map[string]bool{}
	for _, match := range re.FindAllStringSubmatch(blob, -1) {
		code := countryCodeFor(match[1])
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		found = append(found, countryMatch{Code: code, Alias: normalizeText(match[1])})
	}
	return found
}

// Vetoed reports whether the verdict blocks an application.
func (v EligibilityVerdict) Vetoed() bool { return v.Status == EligibilityVeto }

// SignalsJSON encodes the matched phrases for storage.
func (v EligibilityVerdict) SignalsJSON() string {
	b, _ := json.Marshal(v.Signals)
	return string(b)
}

// StatusAfter is the pipeline status the verdict produces for a scored job.
func (v EligibilityVerdict) StatusAfter(score float64) string {
	if v.Vetoed() {
		return StatusRejected
	}
	if score >= autoApplyMinScore() {
		return "queued"
	}
	return "scored"
}

// canonicalCountryNames are the display names offered by the settings page for
// each country code the rules can recognise.
var canonicalCountryNames = map[string]string{
	"us": "United States", "gb": "United Kingdom", "ie": "Ireland", "ca": "Canada",
	"au": "Australia", "nz": "New Zealand", "de": "Germany", "fr": "France",
	"nl": "Netherlands", "be": "Belgium", "es": "Spain", "pt": "Portugal",
	"it": "Italy", "ch": "Switzerland", "at": "Austria", "se": "Sweden",
	"no": "Norway", "dk": "Denmark", "fi": "Finland", "is": "Iceland",
	"pl": "Poland", "cz": "Czechia", "sk": "Slovakia", "hu": "Hungary",
	"ro": "Romania", "bg": "Bulgaria", "gr": "Greece", "hr": "Croatia",
	"rs": "Serbia", "si": "Slovenia", "ee": "Estonia", "lv": "Latvia",
	"lt": "Lithuania", "ua": "Ukraine", "tr": "Türkiye", "ru": "Russia",
	"il": "Israel", "ae": "United Arab Emirates", "sa": "Saudi Arabia",
	"qa": "Qatar", "kw": "Kuwait", "bh": "Bahrain", "om": "Oman",
	"jo": "Jordan", "eg": "Egypt", "ma": "Morocco", "ng": "Nigeria",
	"ke": "Kenya", "za": "South Africa", "gh": "Ghana", "pk": "Pakistan",
	"in": "India", "bd": "Bangladesh", "lk": "Sri Lanka", "np": "Nepal",
	"cn": "China", "hk": "Hong Kong", "tw": "Taiwan", "jp": "Japan",
	"kr": "South Korea", "sg": "Singapore", "my": "Malaysia", "id": "Indonesia",
	"ph": "Philippines", "th": "Thailand", "vn": "Vietnam", "br": "Brazil",
	"ar": "Argentina", "cl": "Chile", "co": "Colombia", "pe": "Peru",
	"mx": "Mexico", "cr": "Costa Rica", "uy": "Uruguay",
}

// CountryCatalogEntry names one country the rules can recognise.
type CountryCatalogEntry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// CountryCatalog lists every country code the rules can detect, sorted by name,
// so the settings page can offer countries by name instead of raw ISO codes.
func CountryCatalog() []CountryCatalogEntry {
	seen := map[string]bool{}
	entries := []CountryCatalogEntry{}
	for _, code := range countryAliases {
		if seen[code] {
			continue
		}
		seen[code] = true
		name := canonicalCountryNames[code]
		if name == "" {
			name = strings.ToUpper(code)
		}
		entries = append(entries, CountryCatalogEntry{Code: code, Name: name})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Code < entries[j].Code
	})
	return entries
}

func on(p *bool) bool { return p != nil && *p }

// EvaluateEligibility applies the operator's policy to one posting. It is pure
// and local — no network call, no LLM — and every verdict quotes the phrases
// that decided it, so a wrong verdict can be traced back to a rule.
//
// Precedence: existing-work-rights veto, then country-bounded remote veto, then
// the sponsored-relocation pass, the worldwide pass, and finally the pass for a
// role bounded to a country the operator already has rights in. A veto always
// beats a pass. A posting with no signal is `unknown` (passed through) unless the
// operator asked for unknown postings to be rejected.
func EvaluateEligibility(job *models.Job, rules *EligibilityRules) EligibilityVerdict {
	policy := rules.WithDefaults()
	if !on(policy.Enabled) {
		return EligibilityVerdict{
			Status:  EligibilityPass,
			Rule:    RuleDisabled,
			Signals: []string{},
			Reason:  "Eligibility rules are off — every posting passes through unfiltered.",
		}
	}
	if job == nil {
		return EligibilityVerdict{
			Status:  EligibilityUnknown,
			Rule:    RuleNoSignal,
			Signals: []string{},
			Reason:  "No posting text to evaluate.",
		}
	}

	blob := normalizeText(job.Title + " " + nullStr(job.Location) + " " + job.Description)
	m := matcher{blob: blob}

	// 1. Roles that need work rights you would have to already hold.
	countryPermitHits, permitCountries := m.matchCountries(policy.WorkPermitCountryPhrase)
	permitHits := unique(append(m.match(policy.WorkPermitPhrases, false), countryPermitHits...))
	if on(policy.VetoLocalWorkPermit) && (len(permitHits) > 0 || len(permitCountries) > 0) {
		evidence := unique(append(append([]string{}, permitHits...), aliases(permitCountries)...))
		reason := "Vetoed: the role requires work rights you would have to already hold"
		if len(permitCountries) > 0 {
			reason += " in " + strings.Join(aliases(permitCountries), ", ")
		}
		if q := quoteList(evidence, 3); q != "" {
			reason += " (" + q + ")"
		}
		return EligibilityVerdict{
			Status:    EligibilityVeto,
			Rule:      RuleLocalWorkPermit,
			Reason:    reason + ".",
			Signals:   evidence,
			Countries: codes(permitCountries),
		}
	}

	// 2. Remote roles bounded to one country.
	boundHits, boundCountries := m.matchCountries(policy.CountryBoundedPhrases)
	literalHits := m.match(literalPhrases(policy.CountryBoundedPhrases), false)
	boundHits = unique(append(boundHits, literalHits...))

	// Boards often put the bound in the location field alone ("Remote (US)",
	// "Remote - Poland"). If exactly one country is named there on a remote
	// posting, treat it as the bound.
	locationBlob := normalizeText(nullStr(job.Location))
	if job.IsRemote || strings.Contains(locationBlob, "remote") {
		if found := detectCountries(locationBlob); len(found) == 1 && !hasCountry(boundCountries, found[0].Code) {
			boundCountries = append(boundCountries, found[0])
		}
	}

	authorized := map[string]bool{}
	for _, c := range policy.WorkAuthorizedCountries {
		authorized[c] = true
	}
	var blocked, allowed []countryMatch
	for _, c := range boundCountries {
		if on(policy.HonorWorkAuthorizedCountries) && authorized[c.Code] {
			allowed = append(allowed, c)
			continue
		}
		blocked = append(blocked, c)
	}

	if on(policy.VetoCountryBoundedRemote) && (len(blocked) > 0 || len(literalHits) > 0) {
		reason := "Vetoed: the role is limited to candidates based in one country"
		if len(blocked) > 0 {
			reason = "Vetoed: the role is limited to candidates based in " + strings.Join(aliases(blocked), ", ")
		}
		if q := quoteList(boundHits, 3); q != "" {
			reason += " (" + q + ")"
		}
		if len(policy.WorkAuthorizedCountries) > 0 {
			reason += ". You are work-authorized in: " + strings.Join(policy.WorkAuthorizedCountries, ", ") + "."
		} else {
			reason += ". No work-authorized countries are configured yet, so every country-bounded role is vetoed."
		}
		return EligibilityVerdict{
			Status:    EligibilityVeto,
			Rule:      RuleCountryBoundedRemote,
			Reason:    reason,
			Signals:   boundHits,
			Countries: codes(blocked),
		}
	}

	// 3. Passes.
	sponsorHits := m.match(policy.SponsorshipPhrases, true)
	relocHits := m.match(policy.RelocationPhrases, true)
	if on(policy.SponsoredRelocationOK) && len(sponsorHits) > 0 && len(relocHits) > 0 {
		signals := unique(append(append([]string{}, sponsorHits...), relocHits...))
		return EligibilityVerdict{
			Status:               EligibilityPass,
			Rule:                 RuleSponsoredRelocation,
			Reason:               "Passed: visa sponsorship and relocation support are offered (" + quoteList(signals, 4) + "). Relocation readiness will be highlighted in the draft.",
			Signals:              signals,
			HighlightsRelocation: true,
		}
	}

	worldwideHits := m.match(policy.WorldwidePhrases, true)
	if on(policy.WorldwideRemoteOK) && len(worldwideHits) > 0 {
		return EligibilityVerdict{
			Status:  EligibilityPass,
			Rule:    RuleWorldwideRemote,
			Reason:  "Passed: open worldwide or hire-anywhere (" + quoteList(worldwideHits, 4) + ").",
			Signals: worldwideHits,
		}
	}

	if len(allowed) > 0 {
		evidence := quoteList(boundHits, 3)
		if evidence == "" {
			evidence = "location field: " + strings.TrimSpace(nullStr(job.Location))
		}
		return EligibilityVerdict{
			Status:    EligibilityPass,
			Rule:      RuleAuthorizedCountry,
			Reason:    "Passed: bounded to " + strings.Join(aliases(allowed), ", ") + ", where you are work-authorized (" + evidence + ").",
			Signals:   boundHits,
			Countries: codes(allowed),
		}
	}

	if on(policy.VetoUnknown) {
		return EligibilityVerdict{
			Status:  EligibilityVeto,
			Rule:    RuleUnknownVetoByOperator,
			Signals: []string{},
			Reason:  "Vetoed: no eligibility signal was found and \"reject when unknown\" is on.",
		}
	}
	return EligibilityVerdict{
		Status:  EligibilityUnknown,
		Rule:    RuleNoSignal,
		Signals: []string{},
		Reason:  "No eligibility signal found — passed through without a veto.",
	}
}

func aliases(matches []countryMatch) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Alias)
	}
	return out
}

func codes(matches []countryMatch) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Code)
	}
	return out
}

func hasCountry(matches []countryMatch, code string) bool {
	for _, m := range matches {
		if m.Code == code {
			return true
		}
	}
	return false
}

// quoteList renders matched phrases for a verdict reason, capped so a long
// description cannot turn a reason into a wall of text.
func quoteList(phrases []string, max int) string {
	var parts []string
	for i, p := range phrases {
		if i >= max {
			parts = append(parts, fmt.Sprintf("+%d more", len(phrases)-max))
			break
		}
		parts = append(parts, `"`+p+`"`)
	}
	return strings.Join(parts, ", ")
}
