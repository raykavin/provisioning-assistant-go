package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"provisioning-assistant/internal/database"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/handler"
	"provisioning-assistant/internal/logger"
	"provisioning-assistant/internal/repository"
	"provisioning-assistant/internal/services"
	"provisioning-assistant/internal/telegram"
	"provisioning-assistant/internal/tl1"
	"provisioning-assistant/internal/unm"

	"github.com/gookit/event"
	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	DatabaseDSN   string
	UNMHost       string
	UNMPort       int
	UNMUsername   string
	UNMPassword   string
	LogLevel      string
}

type Application struct {
	logger       domain.Logger
	db           database.DB
	config       *Config
	services     *Services
	handlers     *Handlers
	eventManager *event.Manager
}

type Services struct {
	Provisioning *services.ProvisioningService
	User         *services.UserService
	Session      *services.SessionService
	ERP          *services.ErpService
}

type Handlers struct {
	Message *handler.MessageHandler
}

// main initializes and runs the provisioning assistant application
func main() {
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("Falha ao inicializar aplicação: %v", err)
	}
	defer app.Close()

	if err := app.Run(); err != nil {
		log.Fatalf("Erro da aplicação: %v", err)
	}
}

// NewApplication creates a new application instance with all dependencies
func NewApplication() (*Application, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Aviso: arquivo .env não encontrado: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar configuração: %w", err)
	}

	logger, err := initializeLogger(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar logger: %w", err)
	}

	db, err := initializeDatabase(config.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar banco de dados: %w", err)
	}

	eventManager := event.NewManager("app")

	services, err := initializeServices(config, db, logger)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar serviços: %w", err)
	}

	handlers := initializeHandlers(services, logger, eventManager)

	app := &Application{
		config:       config,
		logger:       logger,
		db:           db,
		services:     services,
		handlers:     handlers,
		eventManager: eventManager,
	}

	return app, nil
}

// Run starts the application and handles graceful shutdown
func (app *Application) Run() error {
	app.handlers.Message.RegisterEventListeners()

	telegramBot, err := telegram.NewTelegram(app.config.TelegramToken, app.logger, app.eventManager)
	if err != nil {
		return fmt.Errorf("falha ao criar bot do telegram: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app.logStartupMessages()

	telegramBot.Start(ctx)
	return nil
}

// Close performs cleanup operations
func (app *Application) Close() {
	if app.db != nil {
		err := app.db.Close(context.Background())
		if err != nil {
			panic(err)
		}
	}
}

// logStartupMessages displays startup information
func (app *Application) logStartupMessages() {
	app.logger.Info("🤖 Bot iniciado com sucesso!")
	app.logger.Info("📡 Conectado ao UNM em " + app.config.UNMHost)
	app.logger.Info("🗄️ Conectado ao banco de dados")
	app.logger.Info("✅ Pronto para provisionar equipamentos")
}

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	config := &Config{
		TelegramToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		DatabaseDSN:   getEnv("ERP_DATABASE_URL", ""),
		UNMHost:       getEnv("UNM_HOST", ""),
		UNMPort:       getEnvAsInt("UNM_PORT", 3337),
		UNMUsername:   getEnv("UNM_USERNAME", ""),
		UNMPassword:   getEnv("UNM_PASSWORD", ""),
		LogLevel:      getEnv("LOG_LEVEL", "debug"),
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// validateConfig ensures all required configuration values are present
func validateConfig(config *Config) error {
	required := map[string]string{
		"TELEGRAM_BOT_TOKEN": config.TelegramToken,
		"ERP_DATABASE_URL":   config.DatabaseDSN,
		"UNM_HOST":           config.UNMHost,
		"UNM_USERNAME":       config.UNMUsername,
		"UNM_PASSWORD":       config.UNMPassword,
	}

	for key, value := range required {
		if value == "" {
			return fmt.Errorf("variável de ambiente obrigatória %s não está definida", key)
		}
	}

	return nil
}

// initializeLogger creates and configures the application logger
func initializeLogger(logLevel string) (*logger.ZLogXAdapter, error) {
	logConfig := &logger.Config{
		Level:          logLevel,
		DateTimeLayout: "02/01/2006 15:04:05",
		Colored:        true,
		JSONFormat:     false,
		UseEmoji:       true,
	}

	log, err := logger.New(logConfig)
	if err != nil {
		return nil, err
	}

	return &logger.ZLogXAdapter{ZLogX: log}, nil
}

// initializeDatabase creates and connects to the database
func initializeDatabase(dsn string) (*database.PostgresDB, error) {
	ctx := context.Background()
	return database.NewPostgres(ctx, dsn)
}

// initializeServices creates all application services with their dependencies
func initializeServices(config *Config, db database.DB, logger *logger.ZLogXAdapter) (*Services, error) {
	erpRepository := repository.NewErpRepository(db)

	tl1Transport, err := tl1.NewTransport(config.UNMHost, uint16(config.UNMPort))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar transporte TL1: %w", err)
	}

	unmClient := unm.New(config.UNMUsername, config.UNMPassword, tl1Transport, logger)

	services := &Services{
		Provisioning: services.NewProvisioningService(unmClient, logger),
		User:         services.NewUserService(),
		Session:      services.NewSessionService(),
		ERP:          services.NewErpService(erpRepository, logger),
	}

	return services, nil
}

// initializeHandlers creates all application handlers with shared event manager
func initializeHandlers(services *Services, logger *logger.ZLogXAdapter, eventManager *event.Manager) *Handlers {
	return &Handlers{
		Message: handler.NewMessageHandler(
			eventManager,
			services.Provisioning,
			services.User,
			services.Session,
			services.ERP,
			logger,
		),
	}
}

// getEnv retrieves environment variable with fallback to default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves environment variable as integer with fallback
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
