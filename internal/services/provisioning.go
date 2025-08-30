package services

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

type ProvisioningService struct {
}

func NewProvisioningService() *ProvisioningService {
	return &ProvisioningService{}
}

func (s *ProvisioningService) ActivateEquipment(contract, serialNumber string) string {
	// Simulate provisioning process
	log.Printf("Activating equipment - Contract: %s, Serial: %s", contract, serialNumber)

	// Simulate random success/failure
	rand.Seed(time.Now().UnixNano())
	if rand.Float32() > 0.1 { // 90% success rate
		return fmt.Sprintf(
			"✅ Equipamento provisionado com sucesso!\n\n"+
				"📄 Contrato: %s\n"+
				"📟 Serial: %s\n"+
				"📶 Status: ONLINE\n"+
				"🔋 Sinal: -25 dBm\n\n"+
				"O equipamento está pronto para uso!",
			contract, serialNumber,
		)
	}

	return fmt.Sprintf(
		"❌ Falha ao provisionar equipamento\n\n"+
			"📄 Contrato: %s\n"+
			"📟 Serial: %s\n\n"+
			"Por favor, verifique a conexão física e tente novamente.",
		contract, serialNumber,
	)
}

func (s *ProvisioningService) ReplaceEquipment(oldSerial, newSerial, contract string) string {
	// Simulate equipment replacement
	log.Printf("Replacing equipment - Old: %s, New: %s, Contract: %s", oldSerial, newSerial, contract)

	rand.Seed(time.Now().UnixNano())
	if rand.Float32() > 0.1 { // 90% success rate
		return fmt.Sprintf(
			"✅ Troca de equipamento realizada com sucesso!\n\n"+
				"📟 Serial Antigo: %s\n"+
				"📟 Serial Novo: %s\n"+
				"📄 Contrato: %s\n"+
				"📶 Status: ONLINE\n"+
				"🔋 Sinal: -23 dBm\n\n"+
				"A troca foi concluída com sucesso!",
			oldSerial, newSerial, contract,
		)
	}

	return fmt.Sprintf(
		"❌ Falha na troca do equipamento\n\n"+
			"📟 Serial Antigo: %s\n"+
			"📟 Serial Novo: %s\n\n"+
			"Por favor, verifique se o equipamento antigo foi desconectado.",
		oldSerial, newSerial,
	)
}

func (s *ProvisioningService) ChangeAddress(serialNumber, olt, slot, port string) string {
	// Simulate address change
	log.Printf("Changing address - Serial: %s, OLT: %s, Slot: %s, Port: %s",
		serialNumber, olt, slot, port)

	rand.Seed(time.Now().UnixNano())
	if rand.Float32() > 0.1 { // 90% success rate
		return fmt.Sprintf(
			"✅ Mudança de endereço realizada com sucesso!\n\n"+
				"📟 Serial: %s\n"+
				"🌐 Nova OLT: %s\n"+
				"🔌 Slot: %s\n"+
				"🔌 Porta: %s\n"+
				"📶 Status: ONLINE\n"+
				"🔋 Sinal: -24 dBm\n\n"+
				"O equipamento foi reconfigurado no novo endereço!",
			serialNumber, olt, slot, port,
		)
	}

	return fmt.Sprintf(
		"❌ Falha na mudança de endereço\n\n"+
			"📟 Serial: %s\n"+
			"🌐 OLT: %s\n\n"+
			"A porta solicitada pode estar ocupada. Verifique a disponibilidade.",
		serialNumber, olt,
	)
}

func (s *ProvisioningService) ValidateSerial(serial string) bool {
	// Implement serial validation logic
	// For now, just check if it starts with FTTH or GPON
	return len(serial) > 4 &&
		(strings.HasPrefix(serial, "FTTH") ||
			strings.HasPrefix(serial, "GPON") ||
			strings.HasPrefix(serial, "ZTE"))
}

func (s *ProvisioningService) CheckPortAvailability(olt, slot, port string) bool {
	// Simulate port availability check
	// In production, this would check against the actual OLT
	rand.Seed(time.Now().UnixNano())
	return rand.Float32() > 0.2 // 80% of ports are available
}
