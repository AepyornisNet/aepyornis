package model

import (
	"errors"
	"reflect"

	"gorm.io/gorm"
)

type Config struct {
	Model
	EnvConfig  `mapstructure:",squash"`
	UserConfig `mapstructure:",squash"`
}

// UserConfig are options that can be changed at runtime or configured through the web interface
// If they are set through the environment to a non-default value, that will
// take precedence
type UserConfig struct {
	RegistrationDisabled bool `mapstructure:"registration_disabled" json:"registration_disabled"`
	SocialsDisabled      bool `mapstructure:"socials_disabled" json:"socials_disabled"`
}

// EnvConfig are options that are read from the config file or environment only
type EnvConfig struct {
	Host               string `mapstructure:"host" gorm:"-"`                 // The public host name of the server (e.g. example.com)
	Bind               string `mapstructure:"bind" gorm:"-"`                 // Which address to bind to
	WebRoot            string `mapstructure:"web_root" gorm:"-"`             // The web root path (relative to the bind address)
	JWTEncryptionKey   string `mapstructure:"jwt_encryption_key" gorm:"-"`   // Encryption key for JWT
	Dev                bool   `mapstructure:"dev" gorm:"-"`                  // Development mode
	DatabaseDriver     string `mapstructure:"database_driver" gorm:"-"`      // Which database driver to use
	DSN                string `mapstructure:"dsn" gorm:"-"`                  // Database DSN
	Logging            bool   `mapstructure:"logging" gorm:"-"`              // Enable logging
	Debug              bool   `mapstructure:"debug" gorm:"-"`                // Debug logging mode
	Offline            bool   `mapstructure:"offline" gorm:"-"`              // Disable calls to external services
	WorkerDelaySeconds int    `mapstructure:"worker_delay_seconds" gorm:"-"` // Time in seconds between worker runs
	AutoImportEnabled  bool   `mapstructure:"auto_import_enabled" gorm:"-"`  // Enable auto-import scheduler and profile setting
	ActivityPubActive  bool   `mapstructure:"activity_pub_active" gorm:"-"`  // Whether the ActivityPub implementation is active
	VapidPublicKey     string `mapstructure:"vapid_public_key" gorm:"-"`     // Vapid public key used for webpush notifications
	VapidPrivateKey    string `mapstructure:"vapid_private_key" gorm:"-"`    // Vapid private key
	MailjetPublicKey   string `mapstructure:"mailjet_public_key" gorm:"-"`   // Mailjet public key for mail notifications
	MailjetPrivateKey  string `mapstructure:"mailjet_private_key" gorm:"-"`  // Mailjet private key
	MailSenderName     string `mapstructure:"mail_sender_name" gorm:"-"`     // Sender name for notifications
	MailSenderAddress  string `mapstructure:"mail_sender_address" gorm:"-"`  // Sender email address for notifications
	SmtpHost           string `mapstructure:"smtp_host" gorm:"-"`            // SMTP host
	AdminEmail         string `mapstructure:"admin_email" gorm:"-"`          // Email address of the instance owner
	HammerheadClientID string `mapstructure:"hammerhead_client_id" gorm:"-"` // Hammerhead OAuth client id
	HammerheadSecret   string `mapstructure:"hammerhead_client_secret" gorm:"-"`
	HammerheadRedirect string `mapstructure:"hammerhead_redirect_uri" gorm:"-"`
	HammerheadWebhook  string `mapstructure:"hammerhead_webhook_secret" gorm:"-"`

	JWTEncryptionKeyFile string `mapstructure:"jwt_encryption_key_file" gorm:"-"` // File containing the encryption key for JWT
	DSNFile              string `mapstructure:"dsn_file" gorm:"-"`                // File containing the database DSN
}

func getConfig(db *gorm.DB) (*Config, error) {
	var c Config

	if err := db.Model(&Config{}).First(&c).Error; err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Config) UpdateFromDatabase(db *gorm.DB) error {
	dbConfig, err := getConfig(db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	c.copy(dbConfig)

	return nil
}

// copy copies all fields from one Config to another, if the destination field
// is the zero value (false, 0, "")
func (c *Config) copy(from *Config) {
	v := reflect.ValueOf(c).Elem()
	vFrom := reflect.ValueOf(from).Elem()
	n := v.Type().NumField()

	for i := range n {
		if v.Field(i).IsZero() {
			v.Field(i).Set(vFrom.Field(i))
		}
	}
}

func (c *Config) Save(db *gorm.DB) error {
	realCnf, err := getConfig(db)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		c.ID = realCnf.ID
		c.CreatedAt = realCnf.CreatedAt
	}

	return db.Save(c).Error
}
