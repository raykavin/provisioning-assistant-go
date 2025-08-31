package dto

type ConnectionInfo struct {
	AssignmentErpID                 uint64 `db:"assignment_erp_id"`
	AssignmentTitle                 string `db:"assignment_title"`
	ConnectionOltIP                 string `db:"connection_olt_ip"`
	ConnectionOltPort               string `db:"connection_olt_port"`
	ConnectionOltSlot               string `db:"connection_olt_slot"`
	ConnectionEquipmentSerialNumber string `db:"connection_equipment_serial_number"`
	ConnectionClientIP              string `db:"connection_client_ip"`
	ConnectionClientSplitterName    string `db:"connection_client_splitter_name"`
	ConnectionClientSplitterPort    string `db:"connection_client_splitter_port"`
	ConnectionClientPPPoEUsername   string `db:"connection_client_pppoe_username"`
	ConnectionClientPPPoEPassword   string `db:"connection_client_pppoe_password"`
	ConnectionClientVlan            string `db:"connection_client_vlan"`
	ContractDescription             string `db:"contract_description"`
	ClientName                      string `db:"client_name"`
}
