package unm

type OpticalNetworkUnit struct {
	OltID    string
	PonID    string
	OnuNo    string
	Name     string
	Desc     string
	OnuType  string
	IP       string
	AuthType string
	Mac      string
	LoID     string
	Pwd      string
	SwVer    string // Software version
	HwVer    string // Hardware version
}

type OpticalNetworkUnitInfo struct {
	OnuID             string
	RxPower           string
	RxPowerStatus     string
	TxPower           string
	TxPowerStatus     string
	CurrTxBias        string
	CurrTxBiasStatus  string
	Temperature       string
	TemperatureStatus string
	Voltage           string
	VoltageStatus     string
	PTxPower          string
	PRxPower          string
}
