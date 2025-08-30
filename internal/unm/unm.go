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
	// Response patterns
	ErrorPattern    = "EADD=(.*)"
	HeaderLines     = 8
	FooterLines     = -2
	RequiredColumns = 13

	// Command templates
	LoginCommand   = "LOGIN:::CTAG::UN=%s,PWD=%s;"
	LogoutCommand  = "LOGOUT:::CTAG::;"
	OnuListCommand = "LST-ONU::OLTID=%s,PONID=NA-NA-%d-%d:CTAG::;"
	OnuInfoCommand = "LST-OMDDM::OLTID=%s,PONID=NA-NA-%d-%d,ONUIDTYPE=MAC,ONUID=%s:CTAG::;"

	// ONU Provisioning Commands
	DeleteOnuCommand       = "DEL-ONU::OLTID=%s,PONID=NA-NA-%d-%d:CTAG::ONUIDTYPE=MAC,ONUID=%s;"
	AddOnuCommand          = "ADD-ONU::OLTID=%s,PONID=NA-NA-%d-%d:CTAG::AUTHTYPE=MAC,ONUID=%s,NAME=%s | %s - %s,ONUTYPE=%s;"
	SetWanServiceCommand   = "SET-WANSERVICE::OLTID=%s,PONID=NA-NA-%d-%d,ONUIDTYPE=MAC,ONUID=%s:CTAG::STATUS=1,MODE=3,CONNTYPE=2,VLAN=%s,COS=0,QOS=2,NAT=1,IPMODE=3,IPSTACKMODE=1,IP6SRCTYPE=0,PPPOEPROXY=2,PPPOEUSER=%s,PPPOEPASSWD=%s,PPPOENAME=%s,PPPOEMODE=1,%s;"
	ActivateLanPortCommand = "ACT-LANPORT::OLTID=%s,PONID=NA-NA-%d-%d,ONUIDTYPE=MAC,ONUID=%s,ONUPORT=NA-NA-NA-1:CTAG::;"

	// Retry configuration
	MaxRetryAttempts = 3
)

var (
	ErrEmptyHostOrPort    = errors.New("host or port cannot be empty")
	ErrConnectionNotFound = errors.New("connection not established")
	ErrInvalidResponse    = errors.New("invalid response format")
	ErrInsufficientData   = errors.New("insufficient data in response")
	ErrInvalidFormat      = errors.New("invalid response format")
	ErrIllegalSession     = errors.New("illegal session")
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")
	ErrInvalidConfig      = errors.New("invalid provisioning configuration")
)

// Transporter is a interface to transport TCP connection commands
type Transporter interface {
	Close() error
	Reconnect() error
	IsConnected() bool
	Send(ctx context.Context, cmd string) (string, error)
}

// OnuProvisioningConfig contains all parameters needed for ONU provisioning
type OnuProvisioningConfig struct {
	OltIP        string
	PonSlot      uint
	PonPort      uint
	Serial       string
	Splitter     string
	SplitterPort string
	ClientName   string
	Model        string
	Vlan         string
	PPPoEUser    string
	PPPoEPass    string
}

// UNMClient represents a connection to a UNM client with improved error handling and logging.
type UNMClient struct {
	username    string
	password    string
	transporter Transporter
	mtx         sync.Mutex
	connected   bool
	log         domain.Logger

	errorRegex *regexp.Regexp
}

// New creates a new UNMClient instance with pre-compiled regex patterns.
func New(username, password string, transporter Transporter, log domain.Logger) *UNMClient {
	return &UNMClient{
		username:    username,
		password:    password,
		transporter: transporter,
		log:         log,
		errorRegex:  regexp.MustCompile(ErrorPattern),
	}
}

// isIllegalSessionError checks if the error is of type "illegal session"
func (us *UNMClient) isIllegalSessionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "illegal session")
}

// executeWithRetry executes a function with automatic retry in case of illegal session
func (us *UNMClient) executeWithRetry(ctx context.Context, operation func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < MaxRetryAttempts; attempt++ {
		// Ensure connection exists before executing the operation
		if err := us.ensureConnection(ctx); err != nil {
			lastErr = err
			continue
		}

		// Execute the operation
		err := operation(ctx)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// If it's an illegal session error, mark as disconnected and retry
		if us.isIllegalSessionError(err) {
			us.mtx.Lock()
			us.connected = false
			us.mtx.Unlock()

			// If it's not the last attempt, continue the loop
			if attempt < MaxRetryAttempts-1 {
				continue
			}
		} else {
			// If not a session error, don't retry
			return err
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// shakeHand sends a handshake to maintain the keep alive connection
func (us *UNMClient) shakeHand(ctx context.Context) error {
	response, err := us.sendCommand(ctx, "SHAKEHAND:::CTAG::;")
	if err != nil {
		return err
	}

	if err := us.isResponseErr(response); err != nil {
		return err
	}

	return nil
}

// sendCommand is a helper method that combines command sending and error parsing
func (us *UNMClient) sendCommand(ctx context.Context, command string) (string, error) {
	response, err := us.transporter.Send(ctx, command)
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	if err := us.isResponseErr(response); err != nil {
		return "", err
	}

	return response, nil
}

// ensureConnection sends a handshake to check if there's an active connection
// if the connection is down, it reconnects
func (us *UNMClient) ensureConnection(ctx context.Context) error {
	us.mtx.Lock()
	defer us.mtx.Unlock()

	// If already marked as connected, test the connection
	if us.connected {
		return nil
	}

	// First check if we have a connection at transport level
	if !us.transporter.IsConnected() {
		if err := us.reconnectAndLogin(ctx); err != nil {
			return fmt.Errorf("failed to establish connection: %w", err)
		}
		return nil
	}

	// Close existing connection without holding the mutex
	_ = us.close()

	// Reconnect and login
	if err := us.reconnectAndLogin(ctx); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	us.connected = true
	return nil
}

// reconnectAndLogin handles the reconnection and login process
func (us *UNMClient) reconnectAndLogin(ctx context.Context) error {
	if err := us.transporter.Reconnect(); err != nil {
		return fmt.Errorf("reconnect failed: %w", err)
	}

	if err := us.Login(ctx); err != nil {
		us.transporter.Close()
		return fmt.Errorf("login after reconnect failed: %w", err)
	}

	return nil
}

// isResponseErr extracts error information from UNM server response.
func (us *UNMClient) isResponseErr(response string) error {
	if matches := us.errorRegex.FindStringSubmatch(response); len(matches) > 1 {
		errorMsg := strings.TrimSpace(matches[1])
		if errorMsg != "" {
			return fmt.Errorf("UNM server error: %s", errorMsg)
		}
	}

	return nil
}

// Login authenticates with the UNM server.
func (us *UNMClient) Login(ctx context.Context) error {
	command := fmt.Sprintf(LoginCommand, us.username, us.password)

	if _, err := us.sendCommand(ctx, command); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	return nil
}

// Logout logs out from the UNM server.
func (us *UNMClient) Logout(ctx context.Context) error {
	if !us.transporter.IsConnected() {
		return nil // Already disconnected
	}

	if _, err := us.sendCommand(ctx, LogoutCommand); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	return nil
}

// close performs cleanup and close connection
func (us *UNMClient) close() error {
	us.connected = false

	var errs []error

	// Attempt logout first
	if err := us.Logout(context.Background()); err != nil {
		errs = append(errs, err)
	}

	// Close connection if it exists
	if err := us.transporter.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Close gracefully closes the connection to the UNM server.
func (us *UNMClient) Close() error {
	us.mtx.Lock()
	defer us.mtx.Unlock()

	return us.close()
}

// FindAllOpticalNetworkUnits queries all ONUs connected to a specified PON port.
func (us *UNMClient) FindAllOpticalNetworkUnits(
	ctx context.Context,
	olt string,
	ponSlot, ponNumber uint,
	filter string,
) ([]*OpticalNetworkUnit, error) {
	var result []*OpticalNetworkUnit

	err := us.executeWithRetry(ctx, func(ctx context.Context) error {
		command := fmt.Sprintf(OnuListCommand, olt, ponSlot, ponNumber)

		response, err := us.sendCommand(ctx, command)
		if err != nil {
			return fmt.Errorf("failed to query ONUs: %w", err)
		}

		onus, err := us.buildONUsFromResponse(response, filter)
		if err != nil {
			return fmt.Errorf("failed to parse ONU response: %w", err)
		}

		result = onus
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (us *UNMClient) FetchAllOpticalNetworkUnitInformation(
	ctx context.Context,
	ponSlot, ponNumber uint,
	olt, physicalAddr string,
) (*OpticalNetworkUnitInfo, error) {
	var result *OpticalNetworkUnitInfo

	err := us.executeWithRetry(ctx, func(ctx context.Context) error {
		command := fmt.Sprintf(OnuInfoCommand, olt, ponSlot, ponNumber, physicalAddr)

		response, err := us.sendCommand(ctx, command)
		if err != nil {
			return fmt.Errorf("failed to query ONU information: %w", err)
		}

		onuInfo, err := us.buildONUInfoFromResponse(response)
		if err != nil {
			return fmt.Errorf("failed to parse ONU response information: %w", err)
		}

		result = onuInfo
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// OnuProvisioning orchestrates the complete ONU provisioning process
func (us *UNMClient) OnuProvisioning(ctx context.Context, config OnuProvisioningConfig) error {
	// Validate configuration
	if err := us.validateProvisioningConfig(config); err != nil {
		return fmt.Errorf("invalid provisioning configuration: %w", err)
	}

	return us.executeWithRetry(ctx, func(ctx context.Context) error {
		// Step 1: Delete existing ONU (if any)
		if err := us.deleteONU(ctx, config); err != nil {
			// Log but don't fail if deletion fails (ONU might not exist)
			us.log.WithError(err).Debug("ONU deletion failed (may not exist)")
		}

		// Step 2: Add new ONU
		if err := us.addONU(ctx, config); err != nil {
			return fmt.Errorf("failed to add ONU: %w", err)
		}

		// Step 3: Configure WAN services for all ports
		if err := us.configureWanServices(ctx, config); err != nil {
			return fmt.Errorf("failed to configure WAN services: %w", err)
		}

		// Step 4: Activate LAN port
		if err := us.activateLanPort(ctx, config); err != nil {
			return fmt.Errorf("failed to activate LAN port: %w", err)
		}

		us.log.WithFields(map[string]interface{}{
			"olt":    config.OltIP,
			"serial": config.Serial,
			"client": config.ClientName,
		}).Info("ONU provisioning completed successfully")

		return nil
	})
}

// validateProvisioningConfig validates the provisioning configuration
func (us *UNMClient) validateProvisioningConfig(config OnuProvisioningConfig) error {
	if config.OltIP == "" {
		return fmt.Errorf("%w: OLT IP is required", ErrInvalidConfig)
	}
	if config.Serial == "" {
		return fmt.Errorf("%w: Serial (MAC) is required", ErrInvalidConfig)
	}
	if config.Model == "" {
		return fmt.Errorf("%w: Model is required", ErrInvalidConfig)
	}
	if config.Vlan == "" {
		return fmt.Errorf("%w: VLAN is required", ErrInvalidConfig)
	}
	if config.PPPoEUser == "" {
		return fmt.Errorf("%w: PPPoE user is required", ErrInvalidConfig)
	}
	if config.PPPoEPass == "" {
		return fmt.Errorf("%w: PPPoE password is required", ErrInvalidConfig)
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

	us.log.WithFields(map[string]interface{}{
		"olt":    config.OltIP,
		"serial": config.Serial,
	}).Debug("Deleting ONU")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("delete ONU failed: %w", err)
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
		config.Splitter,
		config.SplitterPort,
		config.ClientName,
		config.Model,
	)

	us.log.WithFields(map[string]interface{}{
		"olt":    config.OltIP,
		"serial": config.Serial,
		"client": config.ClientName,
		"model":  config.Model,
	}).Debug("Adding ONU")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("add ONU failed: %w", err)
	}

	return nil
}

// configureWanServices configures WAN services for all ports and SSIDs
func (us *UNMClient) configureWanServices(ctx context.Context, config OnuProvisioningConfig) error {
	// Define port configurations
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
			return fmt.Errorf("failed to configure WAN service for %s: %w", portConfig, err)
		}
	}

	return nil
}

// setWanService configures a single WAN service
func (us *UNMClient) setWanService(ctx context.Context, config OnuProvisioningConfig, portConfig string) error {
	command := fmt.Sprintf(SetWanServiceCommand,
		config.OltIP,
		config.PonSlot,
		config.PonPort,
		config.Serial,
		config.Vlan,
		config.PPPoEUser,
		config.PPPoEPass,
		config.PPPoEUser, // PPPOENAME uses the same value as PPPOEUSER
		portConfig,
	)

	us.log.WithFields(map[string]interface{}{
		"olt":        config.OltIP,
		"serial":     config.Serial,
		"portConfig": portConfig,
		"vlan":       config.Vlan,
	}).Debug("Setting WAN service")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("set WAN service failed: %w", err)
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

	us.log.WithFields(map[string]interface{}{
		"olt":    config.OltIP,
		"serial": config.Serial,
	}).Debug("Activating LAN port")

	_, err := us.sendCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("activate LAN port failed: %w", err)
	}

	return nil
}

// parseResponseLines is a common helper for parsing response lines
func (us *UNMClient) parseResponseLines(response string, minLines int) ([]string, error) {
	formattedResult := strings.ReplaceAll(response, "\r", "")
	lines := splitAndTrimLines(formattedResult)

	if len(lines) <= minLines {
		return nil, ErrInsufficientData
	}

	return lines, nil
}

// buildONUInfoFromResponse parses the ONU information response from the server.
func (us *UNMClient) buildONUInfoFromResponse(response string) (*OpticalNetworkUnitInfo, error) {
	lines, err := us.parseResponseLines(response, HeaderLines)
	if err != nil {
		return nil, fmt.Errorf("optical_info received invalid arguments: %w", err)
	}

	resultLine := lines[HeaderLines : len(lines)+FooterLines]
	if len(resultLine) == 0 {
		return nil, ErrInsufficientData
	}

	items := strings.Split(resultLine[0], "\t")
	if len(items) < RequiredColumns {
		return nil, fmt.Errorf("optical_info command result read buffer does not match: expected %d columns, got %d", RequiredColumns, len(items))
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

// splitAndTrimLines extracts non-empty, trimmed lines from input
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

// buildONUsFromResponse parses the ONU list response from the server.
func (us *UNMClient) buildONUsFromResponse(response string, filter string) ([]*OpticalNetworkUnit, error) {
	lines := strings.Split(response, "\n")
	onus := make([]*OpticalNetworkUnit, 0, len(lines))

	for lineNum, line := range lines {
		if trimmedLine := strings.TrimSpace(line); trimmedLine != "" {
			if onu, err := us.processONULine(trimmedLine, filter); err != nil {
				us.log.WithError(err).WithField("line_number", lineNum).Warn("Failed to parse ONU line")
			} else if onu != nil {
				onus = append(onus, onu)
			}
		}
	}

	return onus, nil
}

// processONULine processes a single ONU line and applies filtering
func (us *UNMClient) processONULine(line, filter string) (*OpticalNetworkUnit, error) {
	attrs := strings.Split(line, "\t")
	if len(attrs) != 12 || attrs[0] == "OLTID" {
		return nil, nil // Skip invalid or header lines
	}

	if filter != "" && !us.matchesFilter(attrs[3], attrs[4], filter) {
		return nil, nil // Skip filtered out items
	}

	return us.createONUFromAttributes(attrs)
}

// matchesFilter checks if name or description matches the filter
func (us *UNMClient) matchesFilter(name, desc, filter string) bool {
	lowerFilter := strings.ToLower(filter)
	lowerName := strings.ToLower(name)
	lowerDesc := strings.ToLower(desc)

	return strings.Contains(lowerName, lowerFilter) || strings.Contains(lowerDesc, lowerFilter)
}

// createONUFromAttributes creates an OpticalNetworkUnit from parsed attributes.
func (us *UNMClient) createONUFromAttributes(attrs []string) (*OpticalNetworkUnit, error) {
	if len(attrs) < 12 {
		return nil, fmt.Errorf("insufficient attributes: expected 12, got %d", len(attrs))
	}

	// Helper function to trim all attributes
	trimmedAttrs := make([]string, len(attrs))
	for i, attr := range attrs {
		trimmedAttrs[i] = strings.TrimSpace(attr)
	}

	return &OpticalNetworkUnit{
		OltID:    trimmedAttrs[0],
		PonID:    trimmedAttrs[1],
		OnuNo:    trimmedAttrs[2],
		Name:     trimmedAttrs[3],
		Desc:     trimmedAttrs[4],
		OnuType:  trimmedAttrs[5],
		IP:       trimmedAttrs[6],
		AuthType: trimmedAttrs[7],
		Mac:      trimmedAttrs[8],
		LoID:     trimmedAttrs[9],
		Pwd:      trimmedAttrs[10],
		SwVer:    trimmedAttrs[11],
	}, nil
}
