package ai

// OccasionOther is used when an event doesn't fit a listed occasion. The
// assistant then goes by the event's title.
const OccasionOther = "other"

const occasionMemorial = "memorial"

// occasionGuidance tells the model how to write for each occasion. The keys
// match the CHECK constraint on events.occasion.
var occasionGuidance = map[string]string{
	"birthday": "This is a birthday. Write birthday wishes. Avoid congratulations unless the " +
		"details call for it, such as a milestone age.",
	"wedding": "This is a wedding. Write to the couple, with love and good wishes for their " +
		"marriage. Congratulations fit here.",
	"anniversary": "This is an anniversary. Celebrate the years together and the relationship.",
	"graduation": "This is a graduation. Congratulate the graduate, and express pride in their " +
		"hard work and excitement for what comes next.",
	"baby": "This is a new baby. Write warm wishes to welcome the baby and to the family. " +
		"Congratulations fit here.",
	"retirement": "This is a retirement. Show appreciation for their career and wish them well " +
		"in the time ahead.",
	"farewell": "This is a farewell: someone is leaving or moving on. Write with appreciation " +
		"for the time together and good wishes for what's next. Avoid congratulations unless " +
		"the details call for it.",
	occasionMemorial: "This is a memorial for someone who has died. Write gently, with love and " +
		"respect, remembering them in the past tense. Never use jokes, celebration, or " +
		"congratulations, and don't try to console with platitudes.",
	OccasionOther: "Work out the occasion from the event title and write what suits it. Avoid " +
		"generic congratulations unless the title calls for it.",
}

// IsOccasion reports whether occasion is one of the known occasions.
func IsOccasion(occasion string) bool {
	_, ok := occasionGuidance[occasion]
	return ok
}

var toneDescriptions = map[string]string{
	"heartfelt": "warm and heartfelt",
	"funny":     "light and funny, with affection, never at their expense",
	"short":     "short and sweet, one sentence each",
	"memory":    "built around a fond memory or what they meant to the sender",
	"gentle":    "short and gentle, one sentence each",
}

// memorialTones are the only tones allowed for a memorial. The web app only
// offers these, and the API enforces it so "funny" can't be requested anyway.
var memorialTones = map[string]bool{"heartfelt": true, "memory": true, "gentle": true}

// toneFor picks the tone to use, falling back to heartfelt for a tone that's
// unknown or doesn't suit the occasion.
func toneFor(occasion, tone string) string {
	if _, ok := toneDescriptions[tone]; !ok {
		return toneDescriptions["heartfelt"]
	}
	if occasion == occasionMemorial && !memorialTones[tone] {
		return toneDescriptions["heartfelt"]
	}
	if occasion != occasionMemorial && tone == "gentle" {
		return toneDescriptions["short"]
	}
	return toneDescriptions[tone]
}
