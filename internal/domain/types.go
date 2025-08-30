// internal/core/domain/types.go
package domain

import "time"

// Events
type MessageEvent struct {
	UserID  int64
	ChatID  int64
	Message string
}

type CallbackEvent struct {
	UserID int64
	ChatID int64
	Data   string
}

// Responses
type MessageResponse struct {
	ChatID   int64
	Text     string
	Keyboard *Keyboard
}

type Keyboard struct {
	Inline  bool
	Buttons [][]Button
}

type Button struct {
	Text string
	Data string
}

// Session states
type SessionState string

const (
	StateIdle              SessionState = "idle"
	StateWaitingCPF        SessionState = "waiting_cpf"
	StateMainMenu          SessionState = "main_menu"
	StateServiceSelection  SessionState = "service_selection"
	StateWaitingContract   SessionState = "waiting_contract"
	StateWaitingSerial     SessionState = "waiting_serial"
	StateConfirmData       SessionState = "confirm_data"
	StateProvisioning      SessionState = "provisioning"
	StateMaintenanceMenu   SessionState = "maintenance_menu"
	StateWaitingOldSerial  SessionState = "waiting_old_serial"
	StateAddressChange     SessionState = "address_change"
	StateWaitingOLT        SessionState = "waiting_olt"
	StateWaitingSlot       SessionState = "waiting_slot"
	StateWaitingPort       SessionState = "waiting_port"
)

// Service types
type ServiceType string

const (
	ServiceActivation    ServiceType = "activation"
	ServiceMaintenance   ServiceType = "maintenance"
	ServiceAddressChange ServiceType = "address_change"
)

// Maintenance types
type MaintenanceType string

const (
	MaintenanceONUChange MaintenanceType = "onu_change"
)

// Session
type Session struct {
	UserID          int64
	ChatID          int64
	State           SessionState
	CPF             string
	UserName        string
	ServiceType     ServiceType
	MaintenanceType MaintenanceType
	Contract        string
	SerialNumber    string
	OldSerialNumber string
	OLT             string
	Slot            string
	Port            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// User
type User struct {
	ID        int64
	CPF       string
	Name      string
	IsValid   bool
	CreatedAt time.Time
}

// Equipment
type Equipment struct {
	SerialNumber string
	Contract     string
	OLT          string
	Slot         string
	Port         string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OLT options
var OLTOptions = []string{
	"PBS - A_MEXICO",
	"PBS - B_HONDURAS_2",
	"PBS - D_DINAMARCA_2",
	"PBS - C_SURINAME",
	"PBS - E_MANILA",
	"PBS - A_AUSTRALIA",
}