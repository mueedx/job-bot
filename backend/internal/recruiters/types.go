package recruiters

// Recruiter is one agency entry shown on /recruiters and exposed via
// /api/recruiters. Fields uses the same track tokens as the resume side
// (fullstack, blockchain, fde, plus any split variants like "react" /
// "node") so the UI can match a recruiter to the user's declared field.
type Recruiter struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Website  string   `json:"website"`
	Countries []string `json:"countries"`
	Fields   []string `json:"fields"`
	Verified bool     `json:"verified"`
}

// RecruiterCatalog is the loadable shape of data/recruiters.yaml. Unknown
// fields are ignored so a hand-edited file never breaks loading.
type RecruiterCatalog struct {
	Recruiters []Recruiter `yaml:"recruiters"`
}
