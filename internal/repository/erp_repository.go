package repository

import (
	"context"
	"errors"
	"provisioning-assistant/internal/database"
	"provisioning-assistant/internal/domain/dto"
)

const getConnInfoQuery = `
SELECT DISTINCT
       a.id AS assignment_erp_id,
       a.title AS assignment_title,
       ai2.ip AS connection_olt_ip,
       as2.port_olt AS connection_olt_port,
       as2.slot_olt AS connection_olt_slot,
       ac.equipment_serial_number AS connection_equipment_serial_number,
       ai3.ip AS connection_client_ip,
       as2.title AS connection_client_splitter_name,
       asp.port AS connection_client_splitter_port,
       ac."user" AS connection_client_pppoe_username,
       ac."password" AS connection_client_pppoe_password,
       ac.vlan AS connection_client_vlan,
       c.description AS contract_description,
       p.name AS client_name
  FROM assignments AS a
 INNER JOIN assignment_incidents AS ai ON a.id = ai.assignment_id
 INNER JOIN contracts AS c ON ai.client_id = c.client_id
 INNER JOIN people AS p ON p.id = c.client_id
 INNER JOIN authentication_contracts AS ac ON c.id = ac.contract_id
  LEFT JOIN authentication_access_points AS acp ON ac.authentication_access_point_id = acp.id
  LEFT JOIN authentication_ips AS ai2 ON acp.authentication_ip_id = ai2.id
  LEFT JOIN authentication_ips AS ai3 ON ac.ip_authentication_id = ai3.id 
  LEFT JOIN authentication_splitter_ports AS asp ON ac.id = asp.authentication_contract_id
  LEFT JOIN authentication_splitters AS as2 ON asp.authentication_splitter_id = as2.id
 WHERE ai.protocol = $1;`

type ErpRepository struct {
	db database.DB
}

// NewErpRepository creates a new ERP repository instance
func NewErpRepository(db database.DB) *ErpRepository {
	if db == nil {
		panic("banco de dados não pode ser nulo")
	}

	return &ErpRepository{
		db: db,
	}
}

// GetConnInfoByProtocol retrieves connection information by protocol number
func (rpt *ErpRepository) GetConnInfoByProtocol(ctx context.Context, protocol string) (*dto.ConnectionInfo, error) {
	if protocol == "" {
		return nil, errors.New("número de protocolo inválido")
	}

	connInfo := &dto.ConnectionInfo{}
	if err := rpt.db.QueryRowStruct(ctx, connInfo, getConnInfoQuery, protocol); err != nil {
		return nil, err
	}

	return connInfo, nil
}
