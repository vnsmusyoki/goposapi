package cashregister

type ActiveRegister struct {
	ID                        string  `json:"id"`
	RegisterNumber            string  `json:"registerNumber"`
	BusinessLocationID        string  `json:"businessLocationId"`
	Status                    string  `json:"status"`
	OpenedBy                  string  `json:"openedBy"`
	OpenedAt                  string  `json:"openedAt"`
	OpeningCashAmount         float64 `json:"openingCashAmount"`
	ExpectedClosingCashAmount float64 `json:"expectedClosingCashAmount"`
}

type PosReadiness struct {
	BusinessLocationID    string
	BusinessLocationName  string
	HasActiveCashRegister bool
	ActiveRegister        *ActiveRegister
	PrinterConfigured     bool
	PrinterTestRequired   bool
	MpesaConfigured       bool
	MpesaStkPushEnabled   bool
	PaymentMethods        []string
	PaymentMethodDetails  []PaymentMethodDefinition
	BlockingReasons       []string
	Warnings              []string
}

type PaymentMethodDefinition struct {
	ID                string `json:"id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	Alias             string `json:"alias"`
	Description       string `json:"description"`
	IsEnabled         bool   `json:"isEnabled"`
	IsCredit          bool   `json:"isCredit"`
	RequiresReference bool   `json:"requiresReference"`
	RequiresPhone     bool   `json:"requiresPhone"`
	SortOrder         int    `json:"sortOrder"`
}

type OpenRegisterInput struct {
	BusinessID         string
	BusinessLocationID string
	OpenedBy           string
	OpeningCashAmount  float64
	Notes              string
}
