package unm

import (
	"context"
	"errors"
	"fmt"
	"provisioning-assistant/internal/domain"
	"regexp"
	"strings"
	"sync"
)

const (
	ErrorPattern    = "EADD=(.*)"
	HeaderLines     = 8
	FooterLines     = -2
	RequiredColumns = 13

	LoginCommand           = "LOGIN:::CTAG::UN=%s,PWD=%s;"
	LogoutCommand          = "LOGOUT:::CTAG::;"
	OnuInfoCommand         = "LST-OMDDM::OLTID=%s,PONID=NA-NA-%d-%d,ONUIDTYPE=MAC,ONUID=%s:CTAG::;"
	DeleteOnuCommand       = "DEL-ONU::OLTID=%s,PONID=NA-NA-%d-%d:CTAG::ONUIDTYPE=MAC,ONUID=%s;"
	AddOnuCommand          = "ADD-ONU::OLTID=%s,PONID=NA-NA-%d-%d:CTAG::AUTHTYPE=MAC,ONUID=%s,NAME=%s | %s - %s,ONUTYPE=%s;"
	SetWanServiceCommand   = "SET-WANSERVICE::OLTID=%s,PONID=NA-NA-%d-%d,ONUIDTYPE=MAC,ONUID=%s:CTAG::STATUS=1,MODE=3,CONNTYPE=2,VLAN=%s,COS=0,QOS=2,NAT=1,IPMODE=3,IPSTACKMODE=1,IP6SRCTYPE=0,PPPOEPROXY=2,PPPOEUSER=%s,PPPOEPASSWD=%s,PPPOENAME=%s,PPPOEMODE=1,%s;"
	ActivateLanPortCommand = "ACT-LANPORT::OLTID=%s,PONID=NA-NA-%d-%d,ONUIDTYPE=MAC,ONUID=%s,ONUPORT=NA-NA-NA-1:CTAG::;"

	MaxRetryAttempts = 3
)

var (
	ErrEmptyHostOrPort          = errors.New("endereço e porta não podem ser vazios")
	ErrConnectionNotEstablished = errors.New("conexão não estabelecida")
	ErrInvalidResponseFormat    = errors.New("formato da resposta inválido")
	ErrInsufficientData         = errors.New("dados insuficientes na resposta")
	ErrIllegalSession           = errors.New("sessão ilegal")
	ErrMaxRetriesExceeded       = errors.New("número máximo de tentativas excedido")
	ErrInvalidConfig            = errors.New("configuração de provisionamento inválida")
)

type Transporter interface {
	Close() error
	Reconnect() error
	IsConnected() bool
	Send(ctx context.Context, cmd string) (string, error)
}

type OnuProvisioningConfig struct {
	OltIP        string
	PonSlot      uint
	PonPort      uint
	Serial       string
	SplitterName string
	SplitterPort string
	ClientName   string
	Model        string
	Vlan         string
	PPPoEUser    string
	PPPoEPass    string
}

type UNMClient struct {
	username    string
	password    string
	transporter Transporter
	mtx         sync.Mutex
	connected   bool
	logger      domain.Logger
	errorRegex  *regexp.Regexp
}

// New creates a new UNM client instance
func New(username, password string, transporter Transporter, logger domain.Logger) *UNMClient {
	return &UNMClient{
		username:    username,
		password:    password,
		logger:      logger,
		transporter: transporter,
		errorRegex:  regexp.MustCompile(ErrorPattern),
	}
}

// Login authenticates with the UNM server
func (us *UNMClient) Login(ctx context.Context) error {
	command := fmt.Sprintf(LoginCommand, us.username, us.password)

	if _, err := us.sendCommand(ctx, command); err != nil {
		return fmt.Errorf("falha no login: %w", err)
	}

	return nil
}

// Logout logs out from the UNM server
func (us *UNMClient) Logout(ctx context.Context) error {
	if !us.transporter.IsConnected() {
		return nil
	}

	if _, err := us.sendCommand(ctx, LogoutCommand); err != nil {
		return fmt.Errorf("falha no logout: %w", err)
	}

	return nil
}

// Close gracefully closes the connection to the UNM server
func (us *UNMClient) Close() error {
	us.mtx.Lock()
	defer us.mtx.Unlock()

	return us.close()
}

// OnuInfo retrieves optical information for a specific ONU
func (us *UNMClient) OnuInfo(ctx context.Context, ponSlot, ponNumber uint, olt, physicalAddr string) (*OpticalNetworkUnitInfo, error) {
	var result *OpticalNetworkUnitInfo

	return result, us.execRetry(ctx, func(ctx context.Context) error {
		command := fmt.Sprintf(OnuInfoCommand, olt, ponSlot, ponNumber, physicalAddr)

		response, err := us.sendCommand(ctx, command)
		if err != nil {
			return fmt.Errorf("falha ao consultar informações da ONU: %w", err)
		}

		onuInfo, err := us.buildONUInfoFromResponse(response)
		if err != nil {
			return fmt.Errorf("falha ao interpretar resposta das informações da ONU: %w", err)
		}

		result = onuInfo
		return nil
	})
}

// OnuProvisioning orchestrates the complete ONU provisioning process
func (us *UNMClient) OnuProvisioning(ctx context.Context, config OnuProvisioningConfig) error {
	if err := us.validateProvisioningConfig(config); err != nil {
		return fmt.Errorf("configuração de provisionamento inválida: %w", err)
	}

	return us.execRetry(ctx, func(ctx context.Context) error {
		if err := us.deleteONU(ctx, config); err != nil {
			us.logger.WithError(err).Debug("Falha ao deletar ONU (pode não existir)")
		}

		if err := us.addONU(ctx, config); err != nil {
			return fmt.Errorf("falha ao adicionar ONU: %w", err)
		}

		if err := us.configureWanServices(ctx, config); err != nil {
			return fmt.Errorf("falha ao configurar serviços WAN: %w", err)
		}

		if err := us.activateLanPort(ctx, config); err != nil {
			return fmt.Errorf("falha ao ativar porta LAN: %w", err)
		}

		us.logger.WithFields(map[string]any{
			"olt":    config.OltIP,
			"serial": config.Serial,
			"client": config.ClientName,
		}).Info("Provisionamento da ONU concluído com sucesso")

		return nil
	})
}

// isIllegalSessionError checks if the error indicates an illegal session
func (us *UNMClient) isIllegalSessionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "illegal session")
}

// execRetry executes an operation with automatic retry on session errors
func (us *UNMClient) execRetry(ctx context.Context, operation func(ctx context.Context) error) error {
	var lastErr error

	for attempt := range MaxRetryAttempts {
		if err := us.ensureConnection(ctx); err != nil {
			lastErr = err
			continue
		}

		err := operation(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		if us.isIllegalSessionError(err) {
			us.mtx.Lock()
			us.connected = false
			us.mtx.Unlock()

			if attempt < MaxRetryAttempts-1 {
				continue
			}
		} else {
			return err
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// sendCommand sends a command to the UNM server and validates the response
func (us *UNMClient) sendCommand(ctx context.Context, command string) (string, error) {
	response, err := us.transporter.Send(ctx, command)
	if err != nil {
		return "", fmt.Errorf("falha no comando: %w", err)
	}

	if err := us.isResponseErr(response); err != nil {
		return "", err
	}

	return response, nil
}

// ensureConnection verifies and establishes connection if needed
func (us *UNMClient) ensureConnection(ctx context.Context) error {
	us.mtx.Lock()
	defer us.mtx.Unlock()

	if us.connected {
		return nil
	}

	if !us.transporter.IsConnected() {
		if err := us.reconnectAndLogin(ctx); err != nil {
			return fmt.Errorf("falha ao estabelecer conexão: %w", err)
		}
		return nil
	}

	_ = us.close()

	if err := us.reconnectAndLogin(ctx); err != nil {
		return fmt.Errorf("falha ao reconectar: %w", err)
	}

	us.connected = true
	return nil
}

// reconnectAndLogin handles the reconnection and login process
func (us *UNMClient) reconnectAndLogin(ctx context.Context) error {
	if err := us.transporter.Reconnect(); err != nil {
		return fmt.Errorf("falha na reconexão: %w", err)
	}

	if err := us.Login(ctx); err != nil {
		us.transporter.Close()
		return fmt.Errorf("falha no login após reconexão: %w", err)
	}

	return nil
}

// isResponseErr checks if the server response contains error information
func (us *UNMClient) isResponseErr(response string) error {
	if matches := us.errorRegex.FindStringSubmatch(response); len(matches) > 1 {
		errorMsg := strings.TrimSpace(matches[1])
		if errorMsg != "" {
			return fmt.Errorf("erro do servidor UNM: %s", errorMsg)
		}
	}

	return nil
}

// close performs cleanup and closes the connection
func (us *UNMClient) close() error {
	us.connected = false

	var errs []error

	if err := us.Logout(context.Background()); err != nil {
		errs = append(errs, err)
	}

	if err := us.transporter.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validateProvisioningConfig validates the ONU provisioning configuration
func (us *UNMClient) validateProvisioningConfig(config OnuProvisioningConfig) error {
	if config.OltIP == "" {
		return fmt.Errorf("%w: IP da OLT é obrigatório", ErrInvalidConfig)
	}
	if config.Serial == "" {
		return fmt.Errorf("%w: endereço físico do equipamento é obrigatório", ErrInvalidConfig)
	}
	if config.Model == "" {
		return fmt.Errorf("%w: modelo é obrigatório", ErrInvalidConfig)
	}
	if config.Vlan == "" {
		return fmt.Errorf("%w: VLAN é obrigatório", ErrInvalidConfig)
	}
	if config.PPPoEUser == "" {
		return fmt.Errorf("%w: usuário PPPoE é obrigatório", ErrInvalidConfig)
	}
	if config.PPPoEPass == "" {
		return fmt.Errorf("%w: senha PPPoE é obrigatório", ErrInvalidConfig)
	}
	return nil
}

// deleteONU removes an existing ONU from the OLT
func (us *UNMClient) deleteONU(ctx context.Context, config OnuProvisioningConfig) error {
	command := fmt.Sprintf(DeleteOnuCommand,
		config.OltIP,
		config.PonSlot,
		config.PonPort,
		config.Serial,
	)

	us.logger.WithFields(map[string]any{
		"olt":    config.OltIP,
		"serial": config.Serial,
	}).Debug("Deletando ONU")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("falha ao deletar ONU: %w", err)
	}

	return nil
}

// addONU adds a new ONU to the OLT
func (us *UNMClient) addONU(ctx context.Context, config OnuProvisioningConfig) error {
	command := fmt.Sprintf(AddOnuCommand,
		config.OltIP,
		config.PonSlot,
		config.PonPort,
		config.Serial,
		config.SplitterName,
		config.SplitterPort,
		config.ClientName,
		config.Model,
	)

	us.logger.WithFields(map[string]any{
		"olt":    config.OltIP,
		"serial": config.Serial,
		"client": config.ClientName,
		"model":  config.Model,
	}).Debug("Adicionando ONU")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("falha ao adicionar ONU: %w", err)
	}

	return nil
}

// configureWanServices configures WAN services for all ports and SSIDs
func (us *UNMClient) configureWanServices(ctx context.Context, config OnuProvisioningConfig) error {
	portConfigs := []string{
		"UPORT=1",
		"UPORT=2",
		"UPORT=3",
		"UPORT=4",
		"SSID=1",
		"SSID=5",
	}

	for _, portConfig := range portConfigs {
		if err := us.setWanService(ctx, config, portConfig); err != nil {
			return fmt.Errorf("falha ao configurar serviço WAN para %s: %w", portConfig, err)
		}
	}

	return nil
}

// setWanService configures a WAN service for a specific port
func (us *UNMClient) setWanService(ctx context.Context, config OnuProvisioningConfig, portConfig string) error {
	command := fmt.Sprintf(SetWanServiceCommand,
		config.OltIP,
		config.PonSlot,
		config.PonPort,
		config.Serial,
		config.Vlan,
		config.PPPoEUser,
		config.PPPoEPass,
		config.PPPoEUser,
		portConfig,
	)

	us.logger.WithFields(map[string]any{
		"olt":        config.OltIP,
		"serial":     config.Serial,
		"portConfig": portConfig,
		"vlan":       config.Vlan,
	}).Debug("Configurando serviço WAN")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("falha ao configurar serviço WAN: %w", err)
	}

	return nil
}

// activateLanPort activates the LAN port on the ONU
func (us *UNMClient) activateLanPort(ctx context.Context, config OnuProvisioningConfig) error {
	command := fmt.Sprintf(ActivateLanPortCommand,
		config.OltIP,
		config.PonSlot,
		config.PonPort,
		config.Serial,
	)

	us.logger.WithFields(map[string]any{
		"olt":    config.OltIP,
		"serial": config.Serial,
	}).Debug("Ativando porta LAN")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("falha ao ativar porta LAN: %w", err)
	}

	return nil
}

// parseResponseLines parses server response and validates minimum line count
func (us *UNMClient) parseResponseLines(response string, minLines int) ([]string, error) {
	formattedResult := strings.ReplaceAll(response, "\r", "")
	lines := splitAndTrimLines(formattedResult)

	if len(lines) <= minLines {
		return nil, ErrInsufficientData
	}

	return lines, nil
}

// buildONUInfoFromResponse parses ONU optical information from server response
func (us *UNMClient) buildONUInfoFromResponse(response string) (*OpticalNetworkUnitInfo, error) {
	lines, err := us.parseResponseLines(response, HeaderLines)
	if err != nil {
		return nil, fmt.Errorf("informações ópticas receberam argumentos inválidos: %w", err)
	}

	resultLine := lines[HeaderLines : len(lines)+FooterLines]
	if len(resultLine) == 0 {
		return nil, ErrInsufficientData
	}

	items := strings.Split(resultLine[0], "\t")
	if len(items) < RequiredColumns {
		return nil, fmt.Errorf("buffer de leitura do resultado do comando optical_info não corresponde: esperado %d colunas, recebido %d", RequiredColumns, len(items))
	}

	return &OpticalNetworkUnitInfo{
		OnuID:             items[0],
		RxPower:           items[1],
		RxPowerStatus:     items[2],
		TxPower:           items[3],
		TxPowerStatus:     items[4],
		CurrTxBias:        items[5],
		CurrTxBiasStatus:  items[6],
		Temperature:       items[7],
		TemperatureStatus: items[8],
		Voltage:           items[9],
		VoltageStatus:     items[10],
		PTxPower:          items[11],
		PRxPower:          items[12],
	}, nil
}

// splitAndTrimLines extracts non-empty, trimmed lines from input string
func splitAndTrimLines(input string) []string {
	lines := strings.Split(input, "\n")
	nonEmptyLines := make([]string, 0, len(lines))

	for _, line := range lines {
		if trimmedLine := strings.TrimSpace(line); trimmedLine != "" {
			nonEmptyLines = append(nonEmptyLines, trimmedLine)
		}
	}

	return nonEmptyLines
}
