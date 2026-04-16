package main

import (
	"strings"

	"salesradar/internal/store"
)

// SuggestedActionProps is the minimal view-model for the suggested-action panel (list drawer + detail page).
type SuggestedActionProps struct {
	Label   string // e.g. "Contact now", "Research first", "Ignore"
	Reason  string // optional 1–2 lines; may be empty
	Variant string // contact | research | ignore — CSS modifier only
}

func suggestedActionFromLead(l store.Lead) SuggestedActionProps {
	label := normalizeSuggestedActionLabel(l)
	return SuggestedActionProps{
		Label:   label,
		Reason:  strings.TrimSpace(suggestedActionReasonFromLead(l, label)),
		Variant: variantFromSuggestedLabel(label),
	}
}

func normalizeSuggestedActionLabel(l store.Lead) string {
	switch strings.TrimSpace(l.Action) {
	case "Contact":
		return "Contact now"
	case "Research first":
		return "Research first"
	case "Ignore":
		return "Ignore"
	}
	if l.SalesReady {
		return "Contact now"
	}
	if strings.TrimSpace(l.LeadStatus) == "discarded" {
		return "Ignore"
	}
	return "Research first"
}

func variantFromSuggestedLabel(label string) string {
	switch label {
	case "Contact now":
		return "contact"
	case "Ignore":
		return "ignore"
	default:
		return "research"
	}
}

func suggestedActionReasonFromLead(l store.Lead, label string) string {
	dup := strings.ToLower(strings.TrimSpace(l.DuplicateStatus))
	isDup := dup == "duplicate" || dup == "suspected_duplicate"

	switch label {
	case "Ignore":
		if isDup {
			return "Possible duplicate or overlap — confirm in your CRM before investing time."
		}
		if strings.TrimSpace(l.ICPMatch) == "no" {
			return "Outside your ideal customer profile with the signals we have."
		}
		if l.PriorityScore > 0 && l.PriorityScore < 40 {
			return "Priority score is low versus your current bar for outreach."
		}
		return ""

	case "Contact now":
		if l.PriorityScore >= 70 {
			return "Strong fit score and readiness — a good moment to reach out."
		}
		if l.DataCompleteness >= 60 {
			return "Enough context on record to start a conversation."
		}
		return ""

	case "Research first":
		var hints []string
		if l.DataCompleteness < 50 && l.DataCompleteness >= 0 {
			hints = append(hints, "profile is still thin")
		}
		if strings.TrimSpace(l.ICPMatch) == "partial" {
			hints = append(hints, "ICP match is only partial")
		}
		for _, r := range l.Reasons {
			rl := strings.ToLower(r)
			if strings.Contains(rl, "weak") && strings.Contains(rl, "signal") {
				hints = append(hints, "training or fit signal needs validation before outreach")
				break
			}
		}
		conf := strings.ToLower(strings.TrimSpace(l.Confidence))
		if conf == "low" {
			hints = append(hints, "confidence in the data is still low")
		}
		if len(hints) > 2 {
			hints = hints[:2]
		}
		if len(hints) > 0 {
			out := strings.Join(hints, "; ")
			if !strings.HasSuffix(out, ".") {
				out += "."
			}
			return truncateRunes(out, 220)
		}
		return "Validate fit and timing on a quick pass before a full outreach push."
	}

	return ""
}
