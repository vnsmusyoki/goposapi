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
	BlockingReasons       []string
	Warnings              []string
}

type OpenRegisterInput struct {
	BusinessID         string
	BusinessLocationID string
	OpenedBy           string
	OpeningCashAmount  float64
	Notes              string
}
