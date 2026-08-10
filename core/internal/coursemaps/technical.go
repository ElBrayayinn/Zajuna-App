package coursemaps

import "regexp"

var technicalActivityPattern = regexp.MustCompile(`(?i)GA\d+-([0-9]+)-AA\d+`)

var transversalCompetencies = map[string]bool{
	"220501046": true,
	"240201524": true,
	"240202501": true,
	"220201501": true,
	"240201528": true,
	"240201064": true,
	"220601501": true,
	"230101507": true,
	"240201526": true,
	"210201501": true,
	"240201529": true,
	"240201501": true,
}

// IsTechnicalActivity follows the checklist rule used by zajuna-sync:
// recognized transversal competency codes are not technical; an activity
// without a recognizable GA/competency code remains eligible because several
// legitimate technical activities use a shortened title.
func IsTechnicalActivity(title string) bool {
	match := technicalActivityPattern.FindStringSubmatch(title)
	return len(match) != 2 || !transversalCompetencies[match[1]]
}
