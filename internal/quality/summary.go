package quality

import (
	"sort"
	"tape-preservation-gate/internal/domain"
)

type ResultSummary struct {
	RuleSetVersion   string         `json:"ruleSetVersion"`
	Total            int            `json:"total"`
	OpenBlocking     int            `json:"openBlocking"`
	PendingWarnings  int            `json:"pendingWarnings"`
	AcceptedWarnings int            `json:"acceptedWarnings"`
	ByRule           map[string]int `json:"byRule"`
	Items            []SummaryItem  `json:"items"`
}

type SummaryItem struct {
	FindingID string               `json:"findingID"`
	RuleCode  string               `json:"ruleCode"`
	Severity  domain.Severity      `json:"severity"`
	Status    domain.FindingStatus `json:"status"`
}

func Summarize(findings []domain.QualityFinding) ResultSummary {
	s := ResultSummary{RuleSetVersion: RuleSetVersion, Total: len(findings), ByRule: map[string]int{}, Items: []SummaryItem{}}
	for _, f := range findings {
		s.ByRule[f.RuleCode]++
		if f.Severity == domain.SeverityBlocking && f.Status != domain.FindingClosed {
			s.OpenBlocking++
		}
		if f.Severity == domain.SeverityWarning {
			if f.Status == domain.FindingAccepted {
				s.AcceptedWarnings++
			} else if f.Status == domain.FindingOpen {
				s.PendingWarnings++
			}
		}
		s.Items = append(s.Items, SummaryItem{FindingID: f.ID, RuleCode: f.RuleCode, Severity: f.Severity, Status: f.Status})
	}
	sort.Slice(s.Items, func(i, j int) bool { return s.Items[i].FindingID < s.Items[j].FindingID })
	return s
}
