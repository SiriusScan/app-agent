package fingerprint

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewCertificateCollector(t *testing.T) {
	t.Log("\n🔍 Testing certificate collector creation...")

	collector := NewCertificateCollector()
	if collector == nil {
		t.Fatal("❌ NewCertificateCollector returned nil")
	}

	if _, ok := collector.(*CertificateCollectorImpl); !ok {
		t.Error("❌ NewCertificateCollector did not return CertificateCollectorImpl")
	}

	t.Log("✅ Certificate collector created successfully")
}

func TestCollectCertificateInfo(t *testing.T) {
	t.Log("\n🔍 Testing certificate information collection...")

	collector := NewCertificateCollector()
	ctx := context.Background()

	info, err := collector.CollectCertificateInfo(ctx)
	if err != nil {
		t.Errorf("❌ CollectCertificateInfo failed: %v", err)
		return
	}

	if info == nil {
		t.Fatal("❌ CollectCertificateInfo returned nil")
	}

	if info.CollectedAt.IsZero() {
		t.Error("❌ CollectedAt should be set")
	}

	// Check that collections were attempted (even if empty)
	if info.SystemCertificates == nil {
		t.Error("❌ SystemCertificates should not be nil")
	}
	if info.UserCertificates == nil {
		t.Error("❌ UserCertificates should not be nil")
	}
	if info.SSLCertificates == nil {
		t.Error("❌ SSLCertificates should not be nil")
	}

	t.Logf("✅ Certificate collection completed - System: %d, User: %d, SSL: %d",
		len(info.SystemCertificates), len(info.UserCertificates), len(info.SSLCertificates))
}

func TestCertificateValidation(t *testing.T) {
	t.Log("\n🔍 Testing certificate validation...")

	collector := &CertificateCollectorImpl{}
	ctx := context.Background()

	// Create test certificates
	validCert := createTestCertificate(t, time.Now().Add(-24*time.Hour), time.Now().Add(365*24*time.Hour))
	expiredCert := createTestCertificate(t, time.Now().Add(-365*24*time.Hour), time.Now().Add(-24*time.Hour))
	futureCert := createTestCertificate(t, time.Now().Add(24*time.Hour), time.Now().Add(365*24*time.Hour))
	expiringSoonCert := createTestCertificate(t, time.Now().Add(-24*time.Hour), time.Now().Add(15*24*time.Hour))

	certs := []*CertificateDetails{validCert, expiredCert, futureCert, expiringSoonCert}

	validations, err := collector.ValidateCertificates(ctx, certs)
	if err != nil {
		t.Errorf("❌ ValidateCertificates failed: %v", err)
		return
	}

	if len(validations) != len(certs) {
		t.Errorf("❌ Expected %d validations, got %d", len(certs), len(validations))
		return
	}

	// Test valid certificate
	if !validations[0].IsValid {
		t.Error("❌ Valid certificate should be marked as valid")
	}
	if validations[0].IsExpired {
		t.Error("❌ Valid certificate should not be marked as expired")
	}

	// Test expired certificate
	if validations[1].IsValid {
		t.Error("❌ Expired certificate should not be marked as valid")
	}
	if !validations[1].IsExpired {
		t.Error("❌ Expired certificate should be marked as expired")
	}
	if len(validations[1].ValidationErrors) == 0 {
		t.Error("❌ Expired certificate should have validation errors")
	}

	// Test future certificate
	if validations[2].IsValid {
		t.Error("❌ Future certificate should not be marked as valid")
	}
	if len(validations[2].ValidationErrors) == 0 {
		t.Error("❌ Future certificate should have validation errors")
	}

	// Test expiring soon certificate
	if !validations[3].IsValid {
		t.Error("❌ Expiring soon certificate should be marked as valid")
	}
	if validations[3].ExpiresInDays >= 30 {
		t.Error("❌ Expiring soon certificate should expire in less than 30 days")
	}
	if len(validations[3].ValidationErrors) == 0 {
		t.Error("❌ Expiring soon certificate should have validation warnings")
	}

	t.Log("✅ Certificate validation working correctly")
}

func TestLoadCertificateFromFile(t *testing.T) {
	t.Log("\n🔍 Testing certificate file loading...")

	collector := &CertificateCollectorImpl{}

	// Create a temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "cert_test")
	if err != nil {
		t.Fatalf("❌ Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test certificate
	cert := createTestCertificate(t, time.Now().Add(-24*time.Hour), time.Now().Add(365*24*time.Hour))

	// Create PEM file
	pemFile := filepath.Join(tempDir, "test.pem")
	if err := createTestPEMFile(pemFile, cert); err != nil {
		t.Fatalf("❌ Failed to create test PEM file: %v", err)
	}

	// Test loading PEM file
	loadedCert, err := collector.loadCertificateFromFile(pemFile, "test")
	if err != nil {
		t.Errorf("❌ Failed to load PEM certificate: %v", err)
		return
	}

	if loadedCert == nil {
		t.Fatal("❌ Loaded certificate is nil")
	}

	if loadedCert.Subject != cert.Subject {
		t.Errorf("❌ Subject mismatch: got %s, want %s", loadedCert.Subject, cert.Subject)
	}

	if loadedCert.FingerprintAlgorithm != "SHA256" {
		t.Errorf("❌ Expected SHA256 fingerprint algorithm, got %s", loadedCert.FingerprintAlgorithm)
	}

	if len(loadedCert.Fingerprint) == 0 {
		t.Error("❌ Fingerprint should not be empty")
	}

	// Test with non-existent file
	_, err = collector.loadCertificateFromFile("/nonexistent/file.pem", "test")
	if err == nil {
		t.Error("❌ Should fail with non-existent file")
	}

	t.Log("✅ Certificate file loading working correctly")
}

func TestLoadCertificatesFromDirectory(t *testing.T) {
	t.Log("\n🔍 Testing certificate directory loading...")

	collector := &CertificateCollectorImpl{}
	ctx := context.Background()

	// Create a temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "cert_dir_test")
	if err != nil {
		t.Fatalf("❌ Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple test certificate files
	cert1 := createTestCertificate(t, time.Now().Add(-24*time.Hour), time.Now().Add(365*24*time.Hour))
	cert2 := createTestCertificate(t, time.Now().Add(-48*time.Hour), time.Now().Add(730*24*time.Hour))

	pemFile1 := filepath.Join(tempDir, "cert1.pem")
	pemFile2 := filepath.Join(tempDir, "cert2.crt")
	txtFile := filepath.Join(tempDir, "readme.txt")

	if err := createTestPEMFile(pemFile1, cert1); err != nil {
		t.Fatalf("❌ Failed to create test PEM file 1: %v", err)
	}
	if err := createTestPEMFile(pemFile2, cert2); err != nil {
		t.Fatalf("❌ Failed to create test PEM file 2: %v", err)
	}
	if err := os.WriteFile(txtFile, []byte("This is not a certificate"), 0644); err != nil {
		t.Fatalf("❌ Failed to create text file: %v", err)
	}

	// Load certificates from directory
	certs, err := collector.loadCertificatesFromDirectory(ctx, tempDir, "test")
	if err != nil {
		t.Errorf("❌ Failed to load certificates from directory: %v", err)
		return
	}

	if len(certs) != 2 {
		t.Errorf("❌ Expected 2 certificates, got %d", len(certs))
		return
	}

	// Verify certificates were loaded correctly
	subjects := make(map[string]bool)
	for _, cert := range certs {
		subjects[cert.Subject] = true
		if cert.StoreLocation != "test" {
			t.Errorf("❌ Expected store location 'test', got %s", cert.StoreLocation)
		}
	}

	if !subjects[cert1.Subject] {
		t.Error("❌ Certificate 1 subject not found in loaded certificates")
	}
	if !subjects[cert2.Subject] {
		t.Error("❌ Certificate 2 subject not found in loaded certificates")
	}

	t.Log("✅ Certificate directory loading working correctly")
}

func TestCertificatePlatformSpecificMethods(t *testing.T) {
	t.Log("\n🔍 Testing platform-specific certificate collection methods...")

	collector := &CertificateCollectorImpl{}
	ctx := context.Background()

	t.Logf("Current platform: %s", runtime.GOOS)

	// Test system certificates
	systemCerts, err := collector.GetSystemCertificates(ctx)
	if err != nil {
		t.Logf("⚠️  System certificate collection failed (may be expected on some platforms): %v", err)
	} else {
		t.Logf("📊 System certificates found: %d", len(systemCerts))
	}

	// Test user certificates
	userCerts, err := collector.GetUserCertificates(ctx)
	if err != nil {
		t.Logf("⚠️  User certificate collection failed (may be expected on some platforms): %v", err)
	} else {
		t.Logf("📊 User certificates found: %d", len(userCerts))
	}

	// Test SSL certificates
	sslCerts, err := collector.GetSSLCertificates(ctx)
	if err != nil {
		t.Logf("⚠️  SSL certificate collection failed: %v", err)
	} else {
		t.Logf("📊 SSL certificates found: %d", len(sslCerts))
	}

	t.Log("✅ Platform-specific methods executed")
}

func TestCertificateToDetails(t *testing.T) {
	t.Log("\n🔍 Testing certificate to details conversion...")

	collector := &CertificateCollectorImpl{}

	// Create test certificate with various features
	template := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject: pkix.Name{
			CommonName:   "Test Certificate",
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
		},
		Issuer: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test CA Org"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:              []string{"example.com", "www.example.com"},
		EmailAddresses:        []string{"test@example.com"},
		BasicConstraintsValid: true,
	}

	// Generate key pair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("❌ Failed to generate private key: %v", err)
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("❌ Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("❌ Failed to parse certificate: %v", err)
	}

	// Convert to details
	details := collector.certificateToDetails(cert, "test-store", "test-location", "/test/path")

	// Verify conversion
	if details.Subject != cert.Subject.String() {
		t.Errorf("❌ Subject mismatch: got %s, want %s", details.Subject, cert.Subject.String())
	}

	if details.Issuer != cert.Issuer.String() {
		t.Errorf("❌ Issuer mismatch: got %s, want %s", details.Issuer, cert.Issuer.String())
	}

	if details.SerialNumber != cert.SerialNumber.String() {
		t.Errorf("❌ Serial number mismatch: got %s, want %s", details.SerialNumber, cert.SerialNumber.String())
	}

	if details.FingerprintAlgorithm != "SHA256" {
		t.Errorf("❌ Expected SHA256 fingerprint algorithm, got %s", details.FingerprintAlgorithm)
	}

	if len(details.Fingerprint) == 0 {
		t.Error("❌ Fingerprint should not be empty")
	}

	// Check key usage conversion
	expectedKeyUsage := []string{"digital_signature", "key_encipherment"}
	if len(details.KeyUsage) != len(expectedKeyUsage) {
		t.Errorf("❌ Key usage count mismatch: got %d, want %d", len(details.KeyUsage), len(expectedKeyUsage))
	}

	// Check extended key usage conversion
	expectedExtKeyUsage := []string{"server_auth", "client_auth"}
	if len(details.ExtendedKeyUsage) != len(expectedExtKeyUsage) {
		t.Errorf("❌ Extended key usage count mismatch: got %d, want %d", len(details.ExtendedKeyUsage), len(expectedExtKeyUsage))
	}

	// Check SAN
	expectedSANCount := len(cert.DNSNames) + len(cert.EmailAddresses)
	if len(details.SubjectAlternativeNames) != expectedSANCount {
		t.Errorf("❌ SAN count mismatch: got %d, want %d", len(details.SubjectAlternativeNames), expectedSANCount)
	}

	if details.Store != "test-store" {
		t.Errorf("❌ Store mismatch: got %s, want test-store", details.Store)
	}

	if details.StoreLocation != "test-location" {
		t.Errorf("❌ Store location mismatch: got %s, want test-location", details.StoreLocation)
	}

	if details.FilePath != "/test/path" {
		t.Errorf("❌ File path mismatch: got %s, want /test/path", details.FilePath)
	}

	t.Log("✅ Certificate to details conversion working correctly")
}

// Helper functions

func createTestCertificate(t *testing.T, notBefore, notAfter time.Time) *CertificateDetails {
	return &CertificateDetails{
		Subject:              "CN=Test Certificate",
		Issuer:               "CN=Test CA",
		SerialNumber:         "12345",
		NotBefore:            notBefore,
		NotAfter:             notAfter,
		Fingerprint:          "abcdef1234567890",
		FingerprintAlgorithm: "SHA256",
		KeyUsage:             []string{"digital_signature", "key_encipherment"},
		Store:                "test",
		StoreLocation:        "test",
	}
}

func createTestPEMFile(filename string, certDetails *CertificateDetails) error {
	// Create a minimal test certificate
	template := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject: pkix.Name{
			CommonName: "Test Certificate",
		},
		Issuer: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             certDetails.NotBefore,
		NotAfter:              certDetails.NotAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	// Generate key pair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return err
	}

	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}

	// Write to file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, pemBlock)
}
