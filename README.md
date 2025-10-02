# 📧 Mail User Management Tool

A feature-rich CLI tool for managing Dovecot mail users with beautiful colored output and comprehensive error handling.

## ✨ Features

- 🎨 **Beautiful colored logging** with indicators
- 🔧 **Build-time configuration** via `config.json`
- 🚀 **Cross-platform support** - builds for 20+ OS/architecture combinations
- 🔐 **Secure password handling** with interactive prompts and masking
- ✅ **Input validation** for email addresses and user existence
- 🛡️ **Safety features** with confirmation prompts for destructive operations
- 📦 **Single binary** with no runtime dependencies (except Dovecot)
- 🎯 **Production-ready** with comprehensive error handling

## 📋 Requirements

- Go 1.16+ (for building)
- Dovecot with `doveadm` command
- Root/sudo privileges (for running)
- `jq` (for building with Makefile)
- Linux/Unix-like system with systemctl

## 🚀 Quick Start

### 1. Clone and Configure

```bash
git clone <repository-url>
cd mail-mgmt
```

### 2. Edit Configuration

Edit `config.json` to match your system:

```json
{
    "Version": "1.0.0",
    "PasswdFile": "/etc/dovecot/passwd",
    "VmailBaseDir": "/var/vmail",
    "DovecotService": "dovecot",
    "VmailUser": "vmail",
    "VmailGroup": "mail",
    "HashScheme": "SHA512-CRYPT",
    "DoveadmCmd": "doveadm"
}
```

### 3. Build

```bash
# Build for your current system
make host_default

# Build for all supported platforms
make all

# Clean build artifacts
make clean
```

The binary will be in `bin/mail-mgmt` (or `bin/mail-mgmt-<os>-<arch>` for cross-compiled builds).

### 4. Install (Optional)

```bash
sudo cp bin/mail-mgmt /usr/local/bin/
sudo chmod +x /usr/local/bin/mail-mgmt
```

## 📖 Usage

### Create a New User

**Interactive (recommended):**
```bash
sudo mail-mgmt create -user john@example.com
```

**Non-interactive:**
```bash
sudo mail-mgmt create -user john@example.com -password "SecurePassword123"
```

**What it does:**
- Validates email format
- Checks if user already exists
- Prompts for password (with confirmation if interactive)
- Generates SHA512-CRYPT hash
- Adds user to `/etc/dovecot/passwd`
- Creates Maildir structure (`/var/vmail/domain/user/Maildir/{cur,new,tmp}`)
- Sets proper ownership and permissions
- Reloads Dovecot service

### Delete a User

**With confirmation:**
```bash
sudo mail-mgmt delete -user john@example.com
```

**Skip confirmation:**
```bash
sudo mail-mgmt delete -user john@example.com -yes
```

**What it does:**
- Validates email format
- Checks if user exists
- Confirms deletion (unless `-yes` flag is used)
- Removes user from `/etc/dovecot/passwd`
- Deletes entire Maildir directory
- Reloads Dovecot service

### Change Password

**Interactive (recommended):**
```bash
sudo mail-mgmt change-password -user john@example.com
# or use the alias:
sudo mail-mgmt passwd -user john@example.com
```

**Non-interactive:**
```bash
sudo mail-mgmt change-password -user john@example.com \
    -old-password "OldPassword" \
    -new-password "NewPassword123"
```

**What it does:**
- Validates email format
- Checks if user exists
- Verifies old password against stored hash
- Prompts for new password (with confirmation if interactive)
- Generates new hash
- Updates `/etc/dovecot/passwd`
- Reloads Dovecot service

### Help and Version

```bash
# Show help
sudo mail-mgmt help

# Show version and build configuration
sudo mail-mgmt version
```

## 🎨 Output Examples

**Success:**
```
ℹ INFO  Creating mail user: john@example.com
→ Enter password for john@example.com:
→ Confirm password:
→ Generating password hash...
→ Adding user to /etc/dovecot/passwd...
→ Creating Maildir structure...
→ Reloading Dovecot...
✓ SUCCESS User created successfully: john@example.com
```

**Error:**
```
✗ ERROR  User already exists: john@example.com
```

**Warning:**
```
⚠ WARNING This will permanently delete user john@example.com and all their emails
```

## 🏗️ Build Configuration

The build process uses `ldflags` to inject configuration from `config.json` at compile time. This means:

- **No runtime configuration files needed** - everything is embedded in the binary
- **Zero-dependency deployment** - just copy the binary

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `Version` | `1.0.0` | Application version |
| `PasswdFile` | `/etc/dovecot/passwd` | Path to Dovecot passwd file |
| `VmailBaseDir` | `/var/vmail` | Base directory for mail storage |
| `DovecotService` | `dovecot` | Systemd service name |
| `VmailUser` | `vmail` | System user for mail files |
| `VmailGroup` | `mail` | System group for mail files |
| `HashScheme` | `SHA512-CRYPT` | Password hashing scheme |
| `DoveadmCmd` | `doveadm` | Path to doveadm command |

### Cross-Platform Builds

The Makefile supports building for multiple platforms:

```bash
# Build for your host platform
make host_default

# Build for specific platform
make linux_arm64

# Build for all platforms
make cross
```

**Supported platforms:**
- Linux: amd64, arm, arm64, ppc64, ppc64le, mips, mipsle, mips64, mips64le, s390x
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- FreeBSD: amd64, 386
- OpenBSD: amd64, 386, arm64
- NetBSD: amd64, 386, arm
- DragonFly BSD: amd64
- Solaris: amd64
- Plan 9: 386, amd64

## 🔧 Development

### Project Structure

```
.
├── main.go           # Main application code
├── config.json       # Build-time configuration
├── Makefile          # Build automation
└── README.md         # This file
```

### TODO

1. **Add a list command:**
```go
case "list":
    handleList()
```

2. **Add export/import for bulk operations:**
```bash
sudo mail-mgmt export > users.csv
sudo mail-mgmt import users.csv
```

3. **Add quota management:**
```bash
sudo mail-mgmt quota -user john@example.com -size 5G
```

4. **Add email alias management:**
```bash
sudo mail-mgmt alias add -from info@example.com -to john@example.com
```

## 🐛 Troubleshooting

### "This program must be run as root"
The tool needs root privileges to modify system files and reload Dovecot.
```bash
sudo mail-mgmt <command>
```

### "doveadm: command not found"
Install Dovecot or update `DoveadmCmd` in `config.json` to the full path.
```bash
# Find doveadm location
which doveadm
```

### "Failed to reload Dovecot"
Check if Dovecot service is running:
```bash
sudo systemctl status dovecot
```

### Build errors with jq
Install `jq` for JSON parsing in Makefile:
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# CentOS/RHEL
sudo yum install jq
```

## 📝 License

[LICENSE](./LICENSE)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 👨‍💻 Author

Jelius Basumatary

## 🙏 Acknowledgments

- Built with Go
- Designed for Dovecot mail server
- Inspired by the need for better mail user management tools
