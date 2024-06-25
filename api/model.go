package api

type Account struct {
	ID           string   `json:"id"`
	SendID       string   `json:"sendId"`
	Balance      int      `json:"balance"`
	CreditLimit  int      `json:"creditLimit"`
	Type         string   `json:"type"`
	CurrencyCode int      `json:"currencyCode"`
	CashbackType string   `json:"cashbackType"`
	MaskedPan    []string `json:"maskedPan"`
	IBAN         string   `json:"iban"`
}

type Jar struct {
	ID           string `json:"id"`
	SendID       string `json:"sendId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	CurrencyCode int    `json:"currencyCode"`
	Balance      int    `json:"balance"`
	Goal         int    `json:"goal"`
}

type Client struct {
	ClientID    string    `json:"clientId"`
	Name        string    `json:"name"`
	WebHookUrl  string    `json:"webHookUrl"`
	Permissions string    `json:"permissions"`
	Accounts    []Account `json:"accounts"`
	Jars        []Jar     `json:"jars"`
}

type Transaction struct {
	ID              string `json:"id"`
	Time            int64  `json:"time"`
	Description     string `json:"description"`
	MCC             int    `json:"mcc"`
	OriginalMCC     int    `json:"originalMcc"`
	Hold            bool   `json:"hold"`
	Amount          int    `json:"amount"`
	OperationAmount int    `json:"operationAmount"`
	CurrencyCode    int    `json:"currencyCode"`
	CommissionRate  int    `json:"commissionRate"`
	CashbackAmount  int    `json:"cashbackAmount"`
	Balance         int    `json:"balance"`
	Comment         string `json:"comment"`
	ReceiptID       string `json:"receiptId"`
	InvoiceID       string `json:"invoiceId"`
	CounterEdrpou   string `json:"counterEdrpou"`
	CounterIban     string `json:"counterIban"`
	CounterName     string `json:"counterName"`
}
