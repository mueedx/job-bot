package recruiters

// Catalog is the shipped list of public recruitment agencies that hire in the
// tracks this bot targets. These are not scraped, never ingested, and never
// written to the jobs table. They feed the /api/recruiters endpoint so the
// dashboard can show "recruiters of your field" grouped by country.
//
// Entries are shipped from the repo so every user gets a usable baseline
// immediately. The user can add, override, or disable entries through
// data/recruiters.yaml, which is merged on top of this catalog at load time.
// Host fields must form a valid URL (https://... or http://...). Fields uses
// the same token vocabulary as the resume tracks: fullstack, blockchain,
// fde, and any split variants.
//
// Where a website could not be verified live, Verified is false and the
// README documents that the entry is unverified. The recruiter URL check
// prints these so they can be fixed or removed before being shown.
var Catalog = []Recruiter{
	// ---- Germany / DACH ----
	{
		ID:       "honeypot-recruiting",
		Name:     "Honeypot Recruiting",
		Website:  "https://www.honeypot.io",
		Countries: []string{"de"},
		Fields:   []string{"fullstack", "backend", "frontend", "devops", "data", "mobile"},
		Verified: true,
	},
	{
		ID:       "moincode",
		Name:     "MoinCode Talent",
		Website:  "https://moin.career",
		Countries: []string{"de"},
		Fields:   []string{"fullstack", "frontend", "backend", "mobile"},
		Verified: false,
	},
	{
		ID:       "motorsoft",
		Name:     "Motorsoft",
		Website:  "https://motorsoft.de",
		Countries: []string{"de"},
		Fields:   []string{"fde", "embedded", "backend", "fullstack"},
		Verified: false,
	},
	// ---- United Kingdom ----
	{
		ID:       "harnham",
		Name:     "Harnham",
		Website:  "https://www.harnham.com",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "frontend", "backend", "data", "mobile", "devops"},
		Verified: true,
	},
	{
		ID:       "revoco",
		Name:     "Revoco",
		Website:  "https://revoco.co.uk",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "frontend", "backend", "fde"},
		Verified: true,
	},
	{
		ID:       "salt-software",
		Name:     "Salt (Remote Tech Recruiting)",
		Website:  "https://www.salt.partners",
		Countries: []string{"gb", "ie"},
		Fields:   []string{"fullstack", "frontend", "backend", "fde", "devops", "data", "mobile"},
		Verified: true,
	},
	{
		ID:       "teksystems-uk",
		Name:     "Teksystems UK",
		Website:  "https://www.teksystems.co.uk",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "data", "mobile", "devops", "fde"},
		Verified: true,
	},
	{
		ID:       "motion-recruitment",
		Name:     "Motion Recruitment",
		Website:  "https://www.motionrecruitment.com",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "fde", "data", "mobile", "devops"},
		Verified: true,
	},
	{
		ID:       "cyber-coders-uk",
		Name:     "CyberCoders UK",
		Website:  "https://www.cybercoders.com/uk",
		Countries: []string{"gb"},
		Fields:   []string{"fde", "backend", "devops", "security", "fullstack"},
		Verified: false,
	},
	{
		ID:       "kforce-uk",
		Name:     "Kforce UK",
		Website:  "https://www.kforce.com",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "fde", "data", "mobile", "devops"},
		Verified: true,
	},
	{
		ID:       "randstad-tech-uk",
		Name:     "Randstad Technologies UK",
		Website:  "https://www.randstad.co.uk",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "data", "mobile", "devops", "fde"},
		Verified: true,
	},
	{
		ID:       "adecco-uk",
		Name:     "Adecco UK (Technology)",
		Website:  "https://www.adecco.co.uk",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "data", "mobile", "devops", "fde"},
		Verified: true,
	},
	{
		ID:       "experis-uk",
		Name:     "Experis UK (Tech & Digital)",
		Website:  "https://www.experis.co.uk",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "data", "mobile", "devops", "fde"},
		Verified: true,
	},
	{
		ID:       "manpower-tech-uk",
		Name:     "Manpower UK (Technology)",
		Website:  "https://www.manpower.co.uk",
		Countries: []string{"gb"},
		Fields:   []string{"fullstack", "backend", "frontend", "data", "mobile", "devops", "fde"},
		Verified: true,
	},
}
