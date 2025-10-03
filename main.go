package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var (
	Version string

	// File paths
	PasswdFile     string
	VmailBaseDir   string
	DovecotService string

	// System user/group for mail
	VmailUser  string
	VmailGroup string

	// Password hashing scheme
	HashScheme string

	// Doveadm command
	DoveadmCmd string

	// Maildir permissions
	MaildirMode = os.FileMode(0700)

	// Colors for logging
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorBold    = "\033[1m"
)

// Logger provides colored logging
type Logger struct {
	useColor bool
}

func NewLogger() *Logger {
	return &Logger{
		useColor: term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	prefix := "ℹ INFO"
	if l.useColor {
		prefix = ColorBlue + "ℹ INFO" + ColorReset
	}
	fmt.Printf(prefix+"  "+format+"\n", args...)
}

func (l *Logger) Success(format string, args ...interface{}) {
	prefix := "✓ SUCCESS"
	if l.useColor {
		prefix = ColorGreen + "✓ SUCCESS" + ColorReset
	}
	fmt.Printf(prefix+" "+format+"\n", args...)
}

func (l *Logger) Warning(format string, args ...interface{}) {
	prefix := "⚠ WARNING"
	if l.useColor {
		prefix = ColorYellow + "⚠ WARNING" + ColorReset
	}
	fmt.Printf(prefix+" "+format+"\n", args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	prefix := "✗ ERROR"
	if l.useColor {
		prefix = ColorRed + "✗ ERROR" + ColorReset
	}
	fmt.Fprintf(os.Stderr, prefix+"  "+format+"\n", args...)
}

func (l *Logger) Step(format string, args ...interface{}) {
	prefix := "→"
	if l.useColor {
		prefix = ColorCyan + "→" + ColorReset
	}
	fmt.Printf(prefix+" "+format+"\n", args...)
}

var log = NewLogger()

func main() {
	// Define subcommands
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	changePassCmd := flag.NewFlagSet("change-password", flag.ExitOnError)

	// Create command flags
	createUser := createCmd.String("user", "", "Email address (user@domain)")
	createPass := createCmd.String("password", "", "Password (optional, will prompt if not provided)")

	// Delete command flags
	deleteUser := deleteCmd.String("user", "", "Email address (user@domain)")
	deleteConfirm := deleteCmd.Bool("yes", false, "Skip confirmation prompt")

	// Change password flags
	changeUser := changePassCmd.String("user", "", "Email address (user@domain)")
	changeOldPass := changePassCmd.String("old-password", "", "Old password (will prompt if not provided)")
	changeNewPass := changePassCmd.String("new-password", "", "New password (will prompt if not provided)")

	// Show help if no arguments
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	// Check for root privileges
	if os.Geteuid() != 0 {
		log.Error("This program must be run as root (use sudo)")
		os.Exit(1)
	}

	// Parse subcommand
	switch os.Args[1] {
	case "create":
		createCmd.Parse(os.Args[2:])
		handleCreate(*createUser, *createPass)
	case "delete":
		deleteCmd.Parse(os.Args[2:])
		handleDelete(*deleteUser, *deleteConfirm)
	case "change-password", "passwd":
		changePassCmd.Parse(os.Args[2:])
		handleChangePassword(*changeUser, *changeOldPass, *changeNewPass)
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		printVersion()
	default:
		log.Error("Unknown command: %s", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	help := `
┌─────────────────────────────────────────────────────────────┐
│          Mail User Management Tool %-17s        │
│          Manage Dovecot mail users with ease                │
└─────────────────────────────────────────────────────────────┘

USAGE:
    sudo mail-mgmt <command> [options]

COMMANDS:
    create              Create a new mail user
    delete              Delete an existing mail user
    change-password     Change user password (alias: passwd)
    help                Show this help message
    version             Show version information

CREATE OPTIONS:
    -user string        Email address (user@domain) [required]
    -password string    Password (optional, will prompt if not provided)

    Example:
        sudo mail-mgmt create -user john@example.com
        sudo mail-mgmt create -user john@example.com -password "SecurePass123"

DELETE OPTIONS:
    -user string        Email address (user@domain) [required]
    -yes                Skip confirmation prompt

    Example:
        sudo mail-mgmt delete -user john@example.com
        sudo mail-mgmt delete -user john@example.com -yes

CHANGE-PASSWORD OPTIONS:
    -user string            Email address (user@domain) [required]
    -old-password string    Current password (will prompt if not provided)
    -new-password string    New password (will prompt if not provided)

    Example:
        sudo mail-mgmt change-password -user john@example.com
        sudo mail-mgmt passwd -user john@example.com

CONFIGURATION:
    Edit the config.json and rebuild the code to customize:
    - PasswdFile: %s
    - VmailBaseDir: %s
    - HashScheme: %s
    - And more...

REQUIREMENTS:
    - Must be run as root (sudo)
    - Dovecot must be installed with doveadm
`
	fmt.Printf(help, "v"+Version, PasswdFile, VmailBaseDir, HashScheme)
}

func printVersion() {
	version := `
Mail User Management Tool v` + Version + `
Built for Dovecot mail server administration
`
	fmt.Println(version)

	fmt.Println("Build-time configuration:")
	fmt.Printf("  PasswdFile:     %s\n", PasswdFile)
	fmt.Printf("  VmailBaseDir:   %s\n", VmailBaseDir)
	fmt.Printf("  DovecotService: %s\n", DovecotService)
	fmt.Printf("  VmailUser:      %s\n", VmailUser)
	fmt.Printf("  VmailGroup:     %s\n", VmailGroup)
	fmt.Printf("  HashScheme:     %s\n", HashScheme)
	fmt.Printf("  DoveadmCmd:     %s\n", DoveadmCmd)
}

func handleCreate(user, password string) {
	log.Info("Creating mail user: %s", user)

	// Validate email
	if !isValidEmail(user) {
		log.Error("Invalid email address: %s", user)
		os.Exit(1)
	}

	// Check if user already exists
	if userExists(user) {
		log.Error("User already exists: %s", user)
		os.Exit(1)
	}

	// Get password if not provided
	if password == "" {
		log.Step("Enter password for %s:", user)
		pass, err := readPassword()
		if err != nil {
			log.Error("Failed to read password: %v", err)
			os.Exit(1)
		}
		log.Step("Confirm password:")
		confirmPass, err := readPassword()
		if err != nil {
			log.Error("Failed to read password: %v", err)
			os.Exit(1)
		}
		if pass != confirmPass {
			log.Error("Passwords do not match")
			os.Exit(1)
		}
		password = pass
	}

	// Generate password hash
	log.Step("Generating password hash...")
	hash, err := generatePasswordHash(password)
	if err != nil {
		log.Error("Failed to generate password hash: %v", err)
		os.Exit(1)
	}

	// Add user to passwd file
	log.Step("Adding user to %s...", PasswdFile)
	if err := addUserToPasswd(user, hash); err != nil {
		log.Error("Failed to add user to passwd file: %v", err)
		os.Exit(1)
	}

	// Create Maildir
	log.Step("Creating Maildir structure...")
	if err := createMaildir(user); err != nil {
		log.Error("Failed to create Maildir: %v", err)
		os.Exit(1)
	}

	// Reload Dovecot
	log.Step("Reloading Dovecot...")
	if err := reloadDovecot(); err != nil {
		log.Warning("Failed to reload Dovecot: %v", err)
	}

	log.Success("User created successfully: %s", user)
}

func handleDelete(user string, skipConfirm bool) {
	log.Info("Deleting mail user: %s", user)

	// Validate email
	if !isValidEmail(user) {
		log.Error("Invalid email address: %s", user)
		os.Exit(1)
	}

	// Check if user exists
	if !userExists(user) {
		log.Error("User does not exist: %s", user)
		os.Exit(1)
	}

	// Confirm deletion
	if !skipConfirm {
		log.Warning("This will permanently delete user %s and all their emails", user)
		fmt.Print("Are you sure? (yes/no): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			log.Info("Deletion cancelled")
			os.Exit(0)
		}
	}

	// Remove from passwd file
	log.Step("Removing user from %s...", PasswdFile)
	if err := removeUserFromPasswd(user); err != nil {
		log.Error("Failed to remove user from passwd file: %v", err)
		os.Exit(1)
	}

	// Delete Maildir
	log.Step("Deleting Maildir...")
	if err := deleteMaildir(user); err != nil {
		log.Warning("Failed to delete Maildir: %v", err)
	}

	// Reload Dovecot
	log.Step("Reloading Dovecot...")
	if err := reloadDovecot(); err != nil {
		log.Warning("Failed to reload Dovecot: %v", err)
	}

	log.Success("User deleted successfully: %s", user)
}

func handleChangePassword(user, oldPass, newPass string) {
	log.Info("Changing password for: %s", user)

	// Validate email
	if !isValidEmail(user) {
		log.Error("Invalid email address: %s", user)
		os.Exit(1)
	}

	// Check if user exists
	if !userExists(user) {
		log.Error("User does not exist: %s", user)
		os.Exit(1)
	}

	// Get old password if not provided
	if oldPass == "" {
		log.Step("Enter current password:")
		pass, err := readPassword()
		if err != nil {
			log.Error("Failed to read password: %v", err)
			os.Exit(1)
		}
		oldPass = pass
	}

	// Verify old password
	log.Step("Verifying current password...")
	storedHash, err := getUserHash(user)
	if err != nil {
		log.Error("Failed to get user hash: %v", err)
		os.Exit(1)
	}

	if !verifyPassword(storedHash, oldPass) {
		log.Error("Current password is incorrect")
		os.Exit(1)
	}

	// Get new password if not provided
	if newPass == "" {
		log.Step("Enter new password:")
		pass, err := readPassword()
		if err != nil {
			log.Error("Failed to read password: %v", err)
			os.Exit(1)
		}
		log.Step("Confirm new password:")
		confirmPass, err := readPassword()
		if err != nil {
			log.Error("Failed to read password: %v", err)
			os.Exit(1)
		}
		if pass != confirmPass {
			log.Error("Passwords do not match")
			os.Exit(1)
		}
		newPass = pass
	}

	// Generate new hash
	log.Step("Generating new password hash...")
	newHash, err := generatePasswordHash(newPass)
	if err != nil {
		log.Error("Failed to generate password hash: %v", err)
		os.Exit(1)
	}

	// Update passwd file
	log.Step("Updating password...")
	if err := updateUserPassword(user, newHash); err != nil {
		log.Error("Failed to update password: %v", err)
		os.Exit(1)
	}

	// Reload Dovecot
	log.Step("Reloading Dovecot...")
	if err := reloadDovecot(); err != nil {
		log.Warning("Failed to reload Dovecot: %v", err)
	}

	log.Success("Password updated successfully for: %s", user)
}

// Utility functions

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func userExists(user string) bool {
	file, err := os.Open(PasswdFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, user+":") {
			return true
		}
	}
	return false
}

func getUserHash(user string) (string, error) {
	file, err := os.Open(PasswdFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, user+":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("user not found")
}

func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

func generatePasswordHash(password string) (string, error) {
	cmd := exec.Command(DoveadmCmd, "pw", "-s", HashScheme, "-p", password)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func verifyPassword(hash, password string) bool {
	cmd := exec.Command(DoveadmCmd, "pw", "-t", hash, "-p", password)
	err := cmd.Run()
	return err == nil
}

func addUserToPasswd(user, hash string) error {
	f, err := os.OpenFile(PasswdFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s:%s\n", user, hash))
	return err
}

func removeUserFromPasswd(user string) error {
	input, err := os.ReadFile(PasswdFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, user+":") {
			newLines = append(newLines, line)
		}
	}

	output := strings.Join(newLines, "\n")
	return os.WriteFile(PasswdFile, []byte(output), 0600)
}

func updateUserPassword(user, newHash string) error {
	input, err := os.ReadFile(PasswdFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, user+":") {
			newLines = append(newLines, fmt.Sprintf("%s:%s", user, newHash))
		} else {
			newLines = append(newLines, line)
		}
	}

	output := strings.Join(newLines, "\n")
	return os.WriteFile(PasswdFile, []byte(output), 0600)
}

func createMaildir(user string) error {
	parts := strings.Split(user, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}
	local, domain := parts[0], parts[1]

	maildir := filepath.Join(VmailBaseDir, domain, local, "Maildir")

	// Create directory structure
	for _, dir := range []string{"cur", "new", "tmp"} {
		path := filepath.Join(maildir, dir)
		if err := os.MkdirAll(path, MaildirMode); err != nil {
			return err
		}
	}

	// Set ownership
	userDir := filepath.Join(VmailBaseDir, domain, local)
	cmd := exec.Command("chown", "-R", VmailUser+":"+VmailGroup, userDir)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set permissions
	cmd = exec.Command("chmod", "-R", "700", userDir)
	return cmd.Run()
}

func deleteMaildir(user string) error {
	parts := strings.Split(user, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}
	local, domain := parts[0], parts[1]

	maildir := filepath.Join(VmailBaseDir, domain, local)

	if _, err := os.Stat(maildir); os.IsNotExist(err) {
		return nil // Already deleted
	}

	return os.RemoveAll(maildir)
}

func reloadDovecot() error {
	cmd := exec.Command("systemctl", "reload", DovecotService)
	return cmd.Run()
}
