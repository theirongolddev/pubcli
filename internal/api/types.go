package api

// SavingsResponse is the top-level response from the Publix savings API.
type SavingsResponse struct {
	Savings                       []SavingItem `json:"Savings"`
	WeeklyAdLatestUpdatedDateTime string       `json:"WeeklyAdLatestUpdatedDateTime"`
	IsPersonalizationEnabled      bool         `json:"IsPersonalizationEnabled"`
	LanguageID                    int          `json:"LanguageId"`
}

// SavingItem represents a single deal/saving from the weekly ad.
type SavingItem struct {
	ID                 string   `json:"id"`
	Title              *string  `json:"title"`
	Description        *string  `json:"description"`
	Savings            *string  `json:"savings"`
	Department         *string  `json:"department"`
	Brand              *string  `json:"brand"`
	Categories         []string `json:"categories"`
	AdditionalDealInfo *string  `json:"additionalDealInfo"`
	ImageURL           *string  `json:"imageUrl"`
	StartFormatted     string   `json:"wa_startDateFormatted"`
	EndFormatted       string   `json:"wa_endDateFormatted"`
}

// StoreResponse is the top-level response from the store locator API.
type StoreResponse struct {
	Stores []Store `json:"Stores"`
}

// Store represents a Publix store location.
type Store struct {
	Key      string `json:"KEY"`
	Name     string `json:"NAME"`
	Addr     string `json:"ADDR"`
	City     string `json:"CITY"`
	State    string `json:"STATE"`
	Zip      string `json:"ZIP"`
	Distance string `json:"DISTANCE"`
	Phone    string `json:"PHONE"`
}
