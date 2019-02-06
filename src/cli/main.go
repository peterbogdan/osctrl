package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

const (
	// Service configuration file
	defConfigFile = "config/tls.json"
	// Application name
	appName = "osctrl-cli"
	// Application version
	appVersion = "0.0.1"
	// Application description
	appDescription = "CLI for osctrl, a fast and efficient operative system management"
	// Application usage
	appUsage = "CLI for osctrl"
)

// Global variables
var (
	db         *gorm.DB
	dbConfig   DBConf
	app        *cli.App
	configFile string
)

// Function to load the configuration file and assign to variables
func loadConfiguration() error {
	// Load file and read config
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	// Backend values
	dbRaw := viper.Sub("db")
	err = dbRaw.Unmarshal(&dbConfig)
	if err != nil {
		return err
	}
	// No errors!
	return nil
}

// Initialization code
func init() {
	// Get path of process
	executableProcess, err := os.Executable()
	if err != nil {
		panic(err)
	}
	configFile = filepath.Dir(executableProcess) + "/" + defConfigFile
	// Initialize CLI details
	app = cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Description = appDescription
	// Flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Value:       configFile,
			Usage:       "Load TLS configuration from `FILE`",
			EnvVar:      "TLS_CONFIG",
			Destination: &configFile,
		},
	}
	// Commands
	app.Commands = []cli.Command{
		{
			Name:  "user",
			Usage: "Commands for users",
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "Add a new user",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "username, u",
							Usage: "Username for the new user",
						},
						cli.StringFlag{
							Name:  "password, p",
							Usage: "Password for the new user",
						},
						cli.BoolFlag{
							Name:   "admin, a",
							Hidden: false,
							Usage:  "Make this user an admin",
						},
						cli.StringFlag{
							Name:  "fullname, n",
							Usage: "Full name for the new user",
						},
					},
					Action: func(c *cli.Context) error {
						// Get values from flags
						username := c.String("username")
						if username == "" {
							fmt.Println("username is required")
							os.Exit(1)
						}
						password := c.String("password")
						fullname := c.String("fullname")
						admin := c.Bool("admin")
						// Create user if it does not exist
						if !userExists(username) {
							newUser := AdminUser{
								Username: username,
								Password: password,
								Fullname: fullname,
								Admin:    admin,
							}
							if err := createUser(newUser); err != nil {
								return err
							}
						}
						return nil
					},
				},
				{
					Name:    "delete",
					Aliases: []string{"d"},
					Usage:   "Delete an existing user",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "username, u",
							Usage: "User to be deleted",
						},
					},
					Action: func(c *cli.Context) error {
						// Get values from flags
						username := c.String("username")
						if username == "" {
							fmt.Println("username is required")
							os.Exit(1)
						}
						return deleteUser(username)
					},
				},
				{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "List all existing users",
					Action: func(c *cli.Context) error {
						users, err := getAllUsers()
						if err != nil {
							return err
						}
						fmt.Printf("Existing users:\n")
						for _, u := range users {
							fmt.Printf("  Username: %s\n", u.Username)
							fmt.Printf("  Fullname: %s\n", u.Fullname)
							fmt.Printf("  Password: %s\n", u.Password)
							fmt.Printf("  Admin? %v\n", u.Admin)
							fmt.Printf("  CSRF: %s\n", u.CSRF)
							fmt.Printf("  Cookie: %s\n", u.Cookie)
							fmt.Printf("  IPAddress: %s\n", u.IPAddress)
							fmt.Printf("  UserAgent: %s\n", u.UserAgent)
							fmt.Println()
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "context",
			Usage: "Commands for TLS context",
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "Add a new TLS context",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "name, n",
							Usage: "Context to be added",
						},
						cli.BoolFlag{
							Name:   "debug, d",
							Hidden: false,
							Usage:  "Context to be added",
						},
						cli.StringFlag{
							Name:  "configuration, conf",
							Usage: "Configuration file to be read",
						},
						cli.StringFlag{
							Name:  "certificate, crt",
							Usage: "Certificate file to be read",
						},
					},
					Action: func(c *cli.Context) error {
						// Get context name
						ctxName := c.String("name")
						if ctxName == "" {
							fmt.Println("Context name is required")
							os.Exit(1)
						}
						// Get configuration
						var configuration string
						confFile := c.String("configuration")
						if confFile != "" {
							configuration = readExternalFile(confFile)
						}
						// Get certificate
						var certificate string
						certFile := c.String("certificate")
						if certFile != "" {
							certificate = readExternalFile(certFile)
						}
						// Create context if it does not exist
						if !contextExists(ctxName) {
							newContext := emptyContext(ctxName)
							newContext.DebugHTTP = c.Bool("debug")
							newContext.Configuration = configuration
							newContext.Certificate = certificate
							if err := createContext(newContext); err != nil {
								return err
							}
						} else {
							fmt.Printf("Context %s already exists!\n", ctxName)
							os.Exit(1)
						}
						return nil
					},
				},
				{
					Name:    "delete",
					Aliases: []string{"d"},
					Usage:   "Delete an existing TLS context",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "name, n",
							Usage: "Context to be deleted",
						},
					},
					Action: func(c *cli.Context) error {
						// Get context name
						ctxName := c.String("name")
						if ctxName == "" {
							fmt.Println("Context name is required")
							os.Exit(1)
						}
						return deleteContext(ctxName)
					},
				},
				{
					Name:    "show",
					Aliases: []string{"s"},
					Usage:   "Show a TLS context",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "name, n",
							Usage: "Context to be displayed",
						},
					},
					Action: func(c *cli.Context) error {
						// Get context name
						ctxName := c.String("name")
						if ctxName == "" {
							fmt.Println("Context name is required")
							os.Exit(1)
						}
						ctx, err := getContext(ctxName)
						if err != nil {
							return err
						}
						fmt.Printf("Context %s\n", ctx.Name)
						fmt.Printf(" Name: %s\n", ctx.Name)
						fmt.Printf(" Secret: %s\n", ctx.Secret)
						fmt.Printf(" SecretPath: %s\n", ctx.SecretPath)
						fmt.Printf(" Type: %v\n", ctx.Type)
						fmt.Printf(" DebugHTTP? %v\n", ctx.DebugHTTP)
						fmt.Printf(" Icon: %s\n", ctx.Icon)
						fmt.Println(" Configuration: ")
						fmt.Printf("%s\n", ctx.Configuration)
						fmt.Println(" Certificate: ")
						fmt.Printf("%s\n", ctx.Certificate)
						fmt.Printf(" EnrollPath: %s\n", ctx.EnrollPath)
						fmt.Printf(" LogPath: %s\n", ctx.LogPath)
						fmt.Printf(" ConfigPath: %s\n", ctx.ConfigPath)
						fmt.Printf(" QueryReadPath: %s\n", ctx.QueryReadPath)
						fmt.Printf(" QueryWritePath: %s\n", ctx.QueryWritePath)
						fmt.Printf(" CarverInitPath: %s\n", ctx.CarverInitPath)
						fmt.Printf(" CarverBlockPath: %s\n", ctx.CarverBlockPath)
						fmt.Println()
						return nil
					},
				},
				{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "List all existing TLS contexts",
					Action: func(c *cli.Context) error {
						contexts, err := getAllContexts()
						if err != nil {
							return err
						}
						fmt.Printf("Existing contexts:\n\n")
						for _, ctx := range contexts {
							fmt.Printf("  Name: %s\n", ctx.Name)

						}
						fmt.Println()
						return nil
					},
				},
			},
		},
		{
			Name:        "node",
			Usage:       "Commands for nodes",
			Subcommands: []cli.Command{},
		},
		{
			Name:        "query",
			Usage:       "Commands for queries",
			Subcommands: []cli.Command{},
		},
	}
	// Load configuration
	err = loadConfiguration()
	if err != nil {
		panic(err)
	}
}

// Go go!
func main() {
	// Database handler
	db = getDB()
	// Close when exit
	defer db.Close()
	// Automigrate tables
	if err := automigrateDB(); err != nil {
		log.Fatalf("Failed to AutoMigrate: %v", err)
	}

	// Let's go!
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
