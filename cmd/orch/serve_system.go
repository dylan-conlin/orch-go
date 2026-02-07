package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/cost"
	"github.com/dylan-conlin/orch-go/pkg/usage"
)

// UsageAPIResponse is the JSON structure returned by /api/usage.
// Note: Percentage fields use *float64 to distinguish between 0% (valid) and unavailable (null).
// When Anthropic API returns null for usage data, these fields will be null in JSON response.
type UsageAPIResponse struct {
	Account         string   `json:"account"`                       // Account email
	AccountName     string   `json:"account_name,omitempty"`        // Account name from accounts.yaml (e.g., "personal", "work")
	FiveHour        *float64 `json:"five_hour_percent"`             // 5-hour session usage % (null if unavailable)
	FiveHourReset   string   `json:"five_hour_reset,omitempty"`     // Human-readable time until 5-hour reset
	Weekly          *float64 `json:"weekly_percent"`                // 7-day weekly usage % (null if unavailable)
	WeeklyReset     string   `json:"weekly_reset,omitempty"`        // Human-readable time until weekly reset
	WeeklyOpus      *float64 `json:"weekly_opus_percent,omitempty"` // 7-day Opus-specific usage % (null if unavailable)
	WeeklyOpusReset string   `json:"weekly_opus_reset,omitempty"`   // Human-readable time until Opus weekly reset
	Error           string   `json:"error,omitempty"`               // Error message if any
}

// CostAPIResponse is the JSON structure returned by /api/usage/cost.
type CostAPIResponse struct {
	CurrentMonthCost float64          `json:"current_month_cost"` // Total cost for current month in USD
	CurrentMonthDate string           `json:"current_month_date"` // YYYY-MM format
	DailyCosts       []cost.DailyCost `json:"daily_costs"`        // Daily costs for last 30 days
	BudgetColor      string           `json:"budget_color"`       // "green", "yellow", or "red" based on budget
	BudgetEmoji      string           `json:"budget_emoji"`       // Emoji for budget status
	Error            string           `json:"error,omitempty"`    // Error message if any
}

// handleUsage returns Claude Max usage stats.
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	info := usage.FetchUsage()

	resp := UsageAPIResponse{}

	if info.Error != "" {
		resp.Error = info.Error
	} else {
		resp.Account = info.Email
		resp.AccountName = lookupAccountName(info.Email)
		if info.FiveHour != nil {
			resp.FiveHour = &info.FiveHour.Utilization
			resp.FiveHourReset = info.FiveHour.TimeUntilReset()
		}
		// else: FiveHour remains nil (JSON: null) indicating data unavailable
		if info.SevenDay != nil {
			resp.Weekly = &info.SevenDay.Utilization
			resp.WeeklyReset = info.SevenDay.TimeUntilReset()
		}
		// else: Weekly remains nil (JSON: null) indicating data unavailable
		if info.SevenDayOpus != nil {
			resp.WeeklyOpus = &info.SevenDayOpus.Utilization
			resp.WeeklyOpusReset = info.SevenDayOpus.TimeUntilReset()
		}
		// else: WeeklyOpus remains nil (JSON: null) indicating data unavailable
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode usage: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleUsageCost returns API cost tracking data.
// Currently returns placeholder data since we need to implement agent cost aggregation.
func (s *Server) handleUsageCost(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// TODO: Implement actual cost aggregation from agent sessions
	// For now, return placeholder data structure
	resp := CostAPIResponse{
		CurrentMonthCost: 0.0,
		CurrentMonthDate: time.Now().UTC().Format("2006-01"),
		DailyCosts:       []cost.DailyCost{},
		BudgetColor:      "green",
		BudgetEmoji:      "🟢",
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode cost data: %v", err), http.StatusInternalServerError)
		return
	}
}

// lookupAccountName finds the account name from ~/.orch/accounts.yaml by matching email.
// Returns the account name (e.g., "personal", "work") if found, empty string otherwise.
func lookupAccountName(email string) string {
	if email == "" {
		return ""
	}

	cfg, err := account.LoadConfig()
	if err != nil {
		return ""
	}

	// Find account by matching email
	for name, acc := range cfg.Accounts {
		if acc.Email == email {
			return name
		}
	}

	return ""
}
