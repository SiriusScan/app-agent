package fingerprint

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CertificateCollectorImpl implements CertificateCollector interface
type CertificateCollectorImpl struct{}

// NewCertificateCollector creates a new certificate collector
func NewCertificateCollector() CertificateCollector {
	return &CertificateCollectorImpl{}
}

// CollectCertificateInfo collects certificate information from all available stores
func (c *CertificateCollectorImpl) CollectCertificateInfo(ctx context.Context) (*CertificateInfo, error) {
	info := &CertificateInfo{
		CollectedAt: time.Now(),
	}

	// Collect system certificates
	systemCerts, err := c.GetSystemCertificates(ctx)
	if err != nil {
		// Don't fail completely, just log the error
		fmt.Printf("Warning: Failed to collect system certificates: %v\n", err)
	}
	info.SystemCertificates = systemCerts

	// Collect user certificates
	userCerts, err := c.GetUserCertificates(ctx)
	if err != nil {
		// Don't fail completely, just log the error
		fmt.Printf("Warning: Failed to collect user certificates: %v\n", err)
	}
	info.UserCertificates = userCerts

	// Collect SSL certificates from common locations
	sslCerts, err := c.GetSSLCertificates(ctx)
	if err != nil {
		// Don't fail completely, just log the error
		fmt.Printf("Warning: Failed to collect SSL certificates: %v\n", err)
	}
	info.SSLCertificates = sslCerts

	// Validate all collected certificates
	allCerts := append(systemCerts, userCerts...)
	allCerts = append(allCerts, sslCerts...)

	if len(allCerts) > 0 {
		validations, err := c.ValidateCertificates(ctx, allCerts)
		if err != nil {
			fmt.Printf("Warning: Failed to validate certificates: %v\n", err)
		}
		info.Validations = validations
	}

	return info, nil
}

// GetSystemCertificates retrieves system certificate store
func (c *CertificateCollectorImpl) GetSystemCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	switch runtime.GOOS {
	case "windows":
		return c.getWindowsSystemCertificates(ctx)
	case "linux":
		return c.getLinuxSystemCertificates(ctx)
	case "darwin":
		return c.getMacOSSystemCertificates(ctx)
	default:
		return []*CertificateDetails{}, nil
	}
}

// GetUserCertificates retrieves user certificate store
func (c *CertificateCollectorImpl) GetUserCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	switch runtime.GOOS {
	case "windows":
		return c.getWindowsUserCertificates(ctx)
	case "linux":
		return c.getLinuxUserCertificates(ctx)
	case "darwin":
		return c.getMacOSUserCertificates(ctx)
	default:
		return []*CertificateDetails{}, nil
	}
}

// GetSSLCertificates retrieves SSL certificates from common locations
func (c *CertificateCollectorImpl) GetSSLCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	var allCerts []*CertificateDetails

	// Common SSL certificate locations
	sslPaths := []string{
		"/etc/ssl/certs",
		"/etc/pki/tls/certs",
		"/usr/local/share/ca-certificates",
		"/System/Library/Keychains",
		"C:\\Windows\\System32\\config\\systemprofile\\AppData\\Roaming\\Microsoft\\SystemCertificates",
	}

	for _, path := range sslPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		certs, err := c.loadCertificatesFromDirectory(ctx, path, "ssl")
		if err != nil {
			// Continue on error, don't fail completely
			fmt.Printf("Warning: Failed to load certificates from %s: %v\n", path, err)
			continue
		}
		allCerts = append(allCerts, certs...)
	}

	return allCerts, nil
}

// ValidateCertificates checks certificate validity and expiration
func (c *CertificateCollectorImpl) ValidateCertificates(ctx context.Context, certs []*CertificateDetails) ([]*CertificateValidation, error) {
	validations := make([]*CertificateValidation, 0, len(certs))
	now := time.Now()

	for _, cert := range certs {
		validation := &CertificateValidation{
			Certificate:   cert,
			ValidatedAt:   now,
			IsExpired:     cert.NotAfter.Before(now),
			ExpiresInDays: int(cert.NotAfter.Sub(now).Hours() / 24),
		}

		// Check if certificate is valid (not expired and not yet valid)
		validation.IsValid = !cert.NotBefore.After(now) && !cert.NotAfter.Before(now)

		// Add validation errors
		var errors []string
		if cert.NotBefore.After(now) {
			errors = append(errors, "Certificate not yet valid")
		}
		if cert.NotAfter.Before(now) {
			errors = append(errors, "Certificate has expired")
		}
		if validation.ExpiresInDays < 30 && validation.ExpiresInDays >= 0 {
			errors = append(errors, fmt.Sprintf("Certificate expires in %d days", validation.ExpiresInDays))
		}

		validation.ValidationErrors = errors
		validations = append(validations, validation)
	}

	return validations, nil
}

// Windows certificate store implementations
func (c *CertificateCollectorImpl) getWindowsSystemCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	// Use PowerShell to query certificate stores
	script := `
		Get-ChildItem -Path 'Cert:\LocalMachine\My', 'Cert:\LocalMachine\Root', 'Cert:\LocalMachine\CA' -Recurse | 
		ForEach-Object {
			[PSCustomObject]@{
				Subject = $_.Subject
				Issuer = $_.Issuer
				SerialNumber = $_.SerialNumber
				NotBefore = $_.NotBefore.ToString('yyyy-MM-ddTHH:mm:ssZ')
				NotAfter = $_.NotAfter.ToString('yyyy-MM-ddTHH:mm:ssZ')
				Thumbprint = $_.Thumbprint
				KeyUsage = ($_.Extensions | Where-Object {$_.Oid.FriendlyName -eq 'Key Usage'}).KeyUsages
				Store = $_.PSParentPath.Split('\')[-1]
			}
		} | ConvertTo-Json -Depth 3
	`

	return c.executePowerShellCertQuery(ctx, script, "system", "machine")
}

func (c *CertificateCollectorImpl) getWindowsUserCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	// Use PowerShell to query user certificate stores
	script := `
		Get-ChildItem -Path 'Cert:\CurrentUser\My', 'Cert:\CurrentUser\Root', 'Cert:\CurrentUser\CA' -Recurse | 
		ForEach-Object {
			[PSCustomObject]@{
				Subject = $_.Subject
				Issuer = $_.Issuer
				SerialNumber = $_.SerialNumber
				NotBefore = $_.NotBefore.ToString('yyyy-MM-ddTHH:mm:ssZ')
				NotAfter = $_.NotAfter.ToString('yyyy-MM-ddTHH:mm:ssZ')
				Thumbprint = $_.Thumbprint
				KeyUsage = ($_.Extensions | Where-Object {$_.Oid.FriendlyName -eq 'Key Usage'}).KeyUsages
				Store = $_.PSParentPath.Split('\')[-1]
			}
		} | ConvertTo-Json -Depth 3
	`

	return c.executePowerShellCertQuery(ctx, script, "user", "currentuser")
}

// Linux certificate store implementations
func (c *CertificateCollectorImpl) getLinuxSystemCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	var allCerts []*CertificateDetails

	// Common Linux certificate directories
	certDirs := []string{
		"/etc/ssl/certs",
		"/etc/pki/ca-trust/extracted/pem",
		"/usr/share/ca-certificates",
		"/etc/ca-certificates",
	}

	for _, dir := range certDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		certs, err := c.loadCertificatesFromDirectory(ctx, dir, "system")
		if err != nil {
			fmt.Printf("Warning: Failed to load certificates from %s: %v\n", dir, err)
			continue
		}
		allCerts = append(allCerts, certs...)
	}

	return allCerts, nil
}

func (c *CertificateCollectorImpl) getLinuxUserCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	var allCerts []*CertificateDetails

	// User certificate directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return allCerts, fmt.Errorf("failed to get user home directory: %w", err)
	}

	userCertDirs := []string{
		filepath.Join(homeDir, ".local/share/ca-certificates"),
		filepath.Join(homeDir, ".config/ca-certificates"),
		filepath.Join(homeDir, ".ssl"),
	}

	for _, dir := range userCertDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		certs, err := c.loadCertificatesFromDirectory(ctx, dir, "user")
		if err != nil {
			fmt.Printf("Warning: Failed to load certificates from %s: %v\n", dir, err)
			continue
		}
		allCerts = append(allCerts, certs...)
	}

	return allCerts, nil
}

// macOS certificate store implementations
func (c *CertificateCollectorImpl) getMacOSSystemCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	// Use security command to dump system keychain
	cmd := exec.CommandContext(ctx, "security", "dump-keychain", "/System/Library/Keychains/SystemRootCertificates.keychain")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative approach with find-certificate
		return c.getMacOSCertificatesAlternative(ctx, "system")
	}

	return c.parseMacOSSecurityOutput(string(output), "system", "system")
}

func (c *CertificateCollectorImpl) getMacOSUserCertificates(ctx context.Context) ([]*CertificateDetails, error) {
	// Use security command to dump user keychain
	cmd := exec.CommandContext(ctx, "security", "dump-keychain", "login.keychain")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative approach with find-certificate
		return c.getMacOSCertificatesAlternative(ctx, "user")
	}

	return c.parseMacOSSecurityOutput(string(output), "user", "login")
}

// Helper methods

func (c *CertificateCollectorImpl) loadCertificatesFromDirectory(ctx context.Context, dir, storeLocation string) ([]*CertificateDetails, error) {
	var certs []*CertificateDetails

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on error
		}

		if d.IsDir() {
			return nil
		}

		// Only process certificate files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".crt" && ext != ".pem" && ext != ".cer" && ext != ".cert" {
			return nil
		}

		cert, err := c.loadCertificateFromFile(path, storeLocation)
		if err != nil {
			// Log error but continue processing
			fmt.Printf("Warning: Failed to load certificate from %s: %v\n", path, err)
			return nil
		}

		if cert != nil {
			certs = append(certs, cert)
		}
		return nil
	})

	return certs, err
}

func (c *CertificateCollectorImpl) loadCertificateFromFile(filePath, storeLocation string) (*CertificateDetails, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Try to decode PEM
	block, _ := pem.Decode(data)
	if block == nil {
		// Try to parse as DER
		cert, err := x509.ParseCertificate(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		return c.certificateToDetails(cert, filepath.Base(filePath), storeLocation, filePath), nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PEM certificate: %w", err)
	}

	return c.certificateToDetails(cert, filepath.Base(filePath), storeLocation, filePath), nil
}

func (c *CertificateCollectorImpl) certificateToDetails(cert *x509.Certificate, store, storeLocation, filePath string) *CertificateDetails {
	// Calculate SHA256 fingerprint
	fingerprint := sha256.Sum256(cert.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	// Convert key usage
	var keyUsage []string
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		keyUsage = append(keyUsage, "digital_signature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		keyUsage = append(keyUsage, "key_encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
		keyUsage = append(keyUsage, "data_encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
		keyUsage = append(keyUsage, "key_agreement")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		keyUsage = append(keyUsage, "cert_sign")
	}

	// Convert extended key usage
	var extKeyUsage []string
	for _, eku := range cert.ExtKeyUsage {
		switch eku {
		case x509.ExtKeyUsageServerAuth:
			extKeyUsage = append(extKeyUsage, "server_auth")
		case x509.ExtKeyUsageClientAuth:
			extKeyUsage = append(extKeyUsage, "client_auth")
		case x509.ExtKeyUsageCodeSigning:
			extKeyUsage = append(extKeyUsage, "code_signing")
		case x509.ExtKeyUsageTimeStamping:
			extKeyUsage = append(extKeyUsage, "time_stamping")
		}
	}

	// Convert SAN
	var san []string
	san = append(san, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		san = append(san, ip.String())
	}
	san = append(san, cert.EmailAddresses...)

	return &CertificateDetails{
		Subject:                 cert.Subject.String(),
		Issuer:                  cert.Issuer.String(),
		SerialNumber:            cert.SerialNumber.String(),
		NotBefore:               cert.NotBefore,
		NotAfter:                cert.NotAfter,
		Fingerprint:             fingerprintHex,
		FingerprintAlgorithm:    "SHA256",
		KeyUsage:                keyUsage,
		ExtendedKeyUsage:        extKeyUsage,
		SubjectAlternativeNames: san,
		Store:                   store,
		StoreLocation:           storeLocation,
		FilePath:                filePath,
	}
}

// Platform-specific helper methods (simplified implementations)

func (c *CertificateCollectorImpl) executePowerShellCertQuery(ctx context.Context, script, storeType, storeLocation string) ([]*CertificateDetails, error) {
	// Execute PowerShell script
	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	_, err := cmd.Output()
	if err != nil {
		return []*CertificateDetails{}, fmt.Errorf("PowerShell execution failed: %w", err)
	}

	// For now, return empty slice - full JSON parsing would be implemented here
	// This is a simplified implementation for the POC
	fmt.Printf("PowerShell certificate query completed for %s store\n", storeType)
	return []*CertificateDetails{}, nil
}

func (c *CertificateCollectorImpl) getMacOSCertificatesAlternative(ctx context.Context, storeType string) ([]*CertificateDetails, error) {
	// Alternative approach using security find-certificate
	var keychain string
	if storeType == "system" {
		keychain = "/System/Library/Keychains/SystemRootCertificates.keychain"
	} else {
		keychain = "login.keychain"
	}

	cmd := exec.CommandContext(ctx, "security", "find-certificate", "-a", "-p", keychain)
	output, err := cmd.Output()
	if err != nil {
		return []*CertificateDetails{}, fmt.Errorf("security command failed: %w", err)
	}

	return c.parseMacOSPEMOutput(string(output), storeType, keychain)
}

func (c *CertificateCollectorImpl) parseMacOSSecurityOutput(output, storeType, storeName string) ([]*CertificateDetails, error) {
	// Simplified parsing - in a full implementation, this would parse the security dump output
	fmt.Printf("Parsing macOS security output for %s store\n", storeType)
	return []*CertificateDetails{}, nil
}

func (c *CertificateCollectorImpl) parseMacOSPEMOutput(output, storeType, storeName string) ([]*CertificateDetails, error) {
	var certs []*CertificateDetails

	// Split output into individual PEM blocks
	pemBlocks := strings.Split(output, "-----END CERTIFICATE-----")

	for i, block := range pemBlocks {
		if !strings.Contains(block, "-----BEGIN CERTIFICATE-----") {
			continue
		}

		// Reconstruct PEM block
		pemData := block + "-----END CERTIFICATE-----"

		pemBlock, _ := pem.Decode([]byte(pemData))
		if pemBlock == nil {
			continue
		}

		cert, err := x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			fmt.Printf("Warning: Failed to parse certificate %d: %v\n", i, err)
			continue
		}

		certDetails := c.certificateToDetails(cert, storeName, storeType, "")
		certs = append(certs, certDetails)
	}

	return certs, nil
}
