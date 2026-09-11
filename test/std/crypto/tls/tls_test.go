package tls_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Certificate fixtures need only exercise TLS parsing and configuration.
const testRSAKeyBits = 1024

func generateTestCert(t *testing.T) ([]byte, []byte) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(keyDER)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}
	parsedRSA, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("Parsed private key has type %T, want *rsa.PrivateKey", parsedKey)
	}
	if !parsedRSA.Equal(priv) {
		t.Fatal("Parsed private key differs from generated key")
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

func TestCipherSuiteName(t *testing.T) {
	tests := []struct {
		id   uint16
		want string
	}{
		{tls.TLS_RSA_WITH_RC4_128_SHA, "TLS_RSA_WITH_RC4_128_SHA"},
		{tls.TLS_RSA_WITH_AES_128_CBC_SHA, "TLS_RSA_WITH_AES_128_CBC_SHA"},
		{0xFFFF, "0xFFFF"},
	}
	for _, tt := range tests {
		got := tls.CipherSuiteName(tt.id)
		if got != tt.want {
			t.Errorf("CipherSuiteName(%#x) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestVersionName(t *testing.T) {
	tests := []struct {
		version uint16
		want    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0xFFFF, "0xFFFF"},
	}
	for _, tt := range tests {
		got := tls.VersionName(tt.version)
		if got != tt.want {
			t.Errorf("VersionName(%#x) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

func TestCipherSuites(t *testing.T) {
	suites := tls.CipherSuites()
	if len(suites) == 0 {
		t.Error("CipherSuites() returned empty slice")
	}
	for _, suite := range suites {
		if suite.Name == "" {
			t.Error("CipherSuite has empty Name")
		}
		if len(suite.SupportedVersions) == 0 {
			t.Errorf("CipherSuite %s has no supported versions", suite.Name)
		}
	}
}

func TestInsecureCipherSuites(t *testing.T) {
	suites := tls.InsecureCipherSuites()
	if len(suites) == 0 {
		t.Error("InsecureCipherSuites() returned empty slice")
	}
	for _, suite := range suites {
		if !suite.Insecure {
			t.Errorf("InsecureCipherSuite %s is not marked as insecure", suite.Name)
		}
	}
}

func TestCertificate(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Error("Certificate has no certificate chain")
	}

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair failed: %v", err)
	}
	if len(loaded.Certificate) != len(cert.Certificate) {
		t.Fatalf("loaded certificate chain length = %d, want %d", len(loaded.Certificate), len(cert.Certificate))
	}
}

func TestDialRejectsUnknownNetwork(t *testing.T) {
	config := &tls.Config{}
	if conn, err := tls.Dial("unknown-network", "unused", config); err == nil {
		conn.Close()
		t.Fatal("Dial unexpectedly accepted an unknown network")
	}
	dialer := &net.Dialer{}
	if conn, err := tls.DialWithDialer(dialer, "unknown-network", "unused", config); err == nil {
		conn.Close()
		t.Fatal("DialWithDialer unexpectedly accepted an unknown network")
	}
}

func TestConfig(t *testing.T) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		ServerName: "example.com",
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Config.MinVersion = %#x, want %#x", config.MinVersion, tls.VersionTLS12)
	}
	if config.ServerName != "example.com" {
		t.Errorf("Config.ServerName = %q, want %q", config.ServerName, "example.com")
	}

	cloned := config.Clone()
	if cloned.ServerName != config.ServerName {
		t.Errorf("Cloned config ServerName = %q, want %q", cloned.ServerName, config.ServerName)
	}
}

func TestConfigSetSessionTicketKeys(t *testing.T) {
	config := &tls.Config{}
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	config.SetSessionTicketKeys([][32]byte{key})
}

func TestClientAuthType(t *testing.T) {
	types := []tls.ClientAuthType{
		tls.NoClientCert,
		tls.RequestClientCert,
		tls.RequireAnyClientCert,
		tls.VerifyClientCertIfGiven,
		tls.RequireAndVerifyClientCert,
	}
	for _, authType := range types {
		if authType < 0 || authType > 4 {
			t.Errorf("Invalid ClientAuthType value: %d", authType)
		}
	}
}

func TestCurveID(t *testing.T) {
	curves := []tls.CurveID{
		tls.CurveP256,
		tls.CurveP384,
		tls.CurveP521,
		tls.X25519,
	}
	for _, curve := range curves {
		if curve == 0 {
			t.Error("CurveID is zero")
		}
	}
}

func TestSignatureScheme(t *testing.T) {
	schemes := []tls.SignatureScheme{
		tls.PKCS1WithSHA256,
		tls.PKCS1WithSHA384,
		tls.PKCS1WithSHA512,
		tls.PSSWithSHA256,
		tls.PSSWithSHA384,
		tls.PSSWithSHA512,
		tls.ECDSAWithP256AndSHA256,
		tls.ECDSAWithP384AndSHA384,
		tls.ECDSAWithP521AndSHA512,
	}
	for _, scheme := range schemes {
		if scheme == 0 {
			t.Error("SignatureScheme is zero")
		}
	}
}

func TestAlertError(t *testing.T) {
	alert := tls.AlertError(42)
	errStr := alert.Error()
	if errStr == "" {
		t.Error("AlertError.Error() returned empty string")
	}
}

func TestNewLRUClientSessionCache(t *testing.T) {
	cache := tls.NewLRUClientSessionCache(10)
	if cache == nil {
		t.Error("NewLRUClientSessionCache returned nil")
	}
}

func TestConnectionState(t *testing.T) {
	var state tls.ConnectionState
	state.Version = tls.VersionTLS13
	state.HandshakeComplete = true
	state.ServerName = "example.com"

	if state.Version != tls.VersionTLS13 {
		t.Errorf("ConnectionState.Version = %#x, want %#x", state.Version, tls.VersionTLS13)
	}
	if !state.HandshakeComplete {
		t.Error("ConnectionState.HandshakeComplete = false, want true")
	}
	if state.ServerName != "example.com" {
		t.Errorf("ConnectionState.ServerName = %q, want example.com", state.ServerName)
	}
}

func TestRenegotiationSupport(t *testing.T) {
	supports := []tls.RenegotiationSupport{
		tls.RenegotiateNever,
		tls.RenegotiateOnceAsClient,
		tls.RenegotiateFreelyAsClient,
	}
	for _, support := range supports {
		if support < 0 || support > 2 {
			t.Errorf("Invalid RenegotiationSupport value: %d", support)
		}
	}
}

func TestQUICEncryptionLevel(t *testing.T) {
	levels := []tls.QUICEncryptionLevel{
		tls.QUICEncryptionLevelInitial,
		tls.QUICEncryptionLevelEarly,
		tls.QUICEncryptionLevelHandshake,
		tls.QUICEncryptionLevelApplication,
	}
	for i, level := range levels {
		if int(level) != i {
			t.Errorf("QUICEncryptionLevel value mismatch: got %d, want %d", level, i)
		}
	}
}

func TestQUICEventKind(t *testing.T) {
	kinds := []tls.QUICEventKind{
		tls.QUICNoEvent,
		tls.QUICSetReadSecret,
		tls.QUICSetWriteSecret,
		tls.QUICWriteData,
		tls.QUICTransportParameters,
		tls.QUICTransportParametersRequired,
		tls.QUICRejectedEarlyData,
		tls.QUICHandshakeDone,
	}
	for _, kind := range kinds {
		if kind < 0 {
			t.Errorf("Invalid QUICEventKind value: %d", kind)
		}
	}
}

func TestClientHelloInfo(t *testing.T) {
	info := &tls.ClientHelloInfo{
		ServerName:   "example.com",
		CipherSuites: []uint16{tls.TLS_AES_128_GCM_SHA256},
	}

	if info.ServerName != "example.com" {
		t.Errorf("ClientHelloInfo.ServerName = %q, want example.com", info.ServerName)
	}

	supported := info.SupportsCertificate(&tls.Certificate{})
	if supported != nil {
		t.Logf("SupportsCertificate returned: %v", supported)
	}
}

func TestCertificateRequestInfo(t *testing.T) {
	info := &tls.CertificateRequestInfo{
		AcceptableCAs: [][]byte{{0x01, 0x02}},
		Version:       tls.VersionTLS13,
	}

	if info.Version != tls.VersionTLS13 {
		t.Errorf("CertificateRequestInfo.Version = %#x, want %#x", info.Version, tls.VersionTLS13)
	}

	supported := info.SupportsCertificate(&tls.Certificate{})
	if supported != nil {
		t.Logf("SupportsCertificate returned: %v", supported)
	}
}

func TestCertificateVerificationError(t *testing.T) {
	err := &tls.CertificateVerificationError{
		UnverifiedCertificates: []*x509.Certificate{},
		Err:                    x509.UnknownAuthorityError{},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("CertificateVerificationError.Error() returned empty string")
	}

	unwrapped := err.Unwrap()
	if unwrapped == nil {
		t.Error("CertificateVerificationError.Unwrap() returned nil")
	}
}

func TestDialer(t *testing.T) {
	dialer := &tls.Dialer{
		Config: &tls.Config{
			ServerName: "example.com",
		},
	}

	if dialer.Config.ServerName != "example.com" {
		t.Errorf("Dialer.Config.ServerName = %q, want example.com", dialer.Config.ServerName)
	}
}

func TestQUICConfig(t *testing.T) {
	config := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}

	if config.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("QUICConfig.TLSConfig.MinVersion = %#x, want %#x", config.TLSConfig.MinVersion, tls.VersionTLS13)
	}
}

func TestQUICClient(t *testing.T) {
	config := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			ServerName: "example.com",
			MinVersion: tls.VersionTLS13,
		},
	}
	conn := tls.QUICClient(config)
	if conn == nil {
		t.Error("QUICClient returned nil")
	}
}

func TestQUICServer(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		},
	}
	conn := tls.QUICServer(config)
	if conn == nil {
		t.Error("QUICServer returned nil")
	}
}

func TestQUICConn(t *testing.T) {
	config := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			ServerName: "example.com",
			MinVersion: tls.VersionTLS13,
		},
	}
	conn := tls.QUICClient(config)

	event := conn.NextEvent()
	if event.Kind < 0 {
		t.Error("QUICConn.NextEvent() returned invalid event kind")
	}

	conn.Close()
}

func TestRecordHeaderError(t *testing.T) {
	err := tls.RecordHeaderError{
		Msg:          "test error",
		RecordHeader: [5]byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("RecordHeaderError.Error() returned empty string")
	}
}

func TestECHRejectionError(t *testing.T) {
	err := &tls.ECHRejectionError{
		RetryConfigList: []byte{0x01, 0x02, 0x03},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("ECHRejectionError.Error() returned empty string")
	}
}

func TestClientSessionState(t *testing.T) {
	state := &tls.ClientSessionState{}

	if state != nil {
		t.Logf("ClientSessionState created successfully")
	}
}

func TestNewResumptionState(t *testing.T) {
	sessionState := &tls.SessionState{}
	ticket := []byte("test-ticket")

	_, err := tls.NewResumptionState(ticket, sessionState)
	if err != nil {
		t.Logf("NewResumptionState returned error: %v", err)
	}
}

func TestSessionState(t *testing.T) {
	state := &tls.SessionState{
		EarlyData: true,
		Extra:     [][]byte{{0x01, 0x02}},
	}

	if !state.EarlyData {
		t.Error("SessionState.EarlyData = false, want true")
	}

	bytes, err := state.Bytes()
	if err != nil {
		t.Logf("SessionState.Bytes() error: %v", err)
	}
	if len(bytes) == 0 {
		t.Error("SessionState.Bytes() returned empty slice")
	}

	_, err = tls.ParseSessionState(bytes)
	if err != nil {
		t.Logf("ParseSessionState error: %v", err)
	}
}

func TestQUICSessionTicketOptions(t *testing.T) {
	opts := &tls.QUICSessionTicketOptions{
		EarlyData: true,
	}

	if !opts.EarlyData {
		t.Error("QUICSessionTicketOptions.EarlyData = false, want true")
	}
}

func TestQUICEvent(t *testing.T) {
	event := tls.QUICEvent{
		Kind:  tls.QUICSetReadSecret,
		Level: tls.QUICEncryptionLevelHandshake,
		Data:  []byte{0x01, 0x02},
	}

	if event.Kind != tls.QUICSetReadSecret {
		t.Errorf("QUICEvent.Kind = %v, want %v", event.Kind, tls.QUICSetReadSecret)
	}
	if event.Level != tls.QUICEncryptionLevelHandshake {
		t.Errorf("QUICEvent.Level = %v, want %v", event.Level, tls.QUICEncryptionLevelHandshake)
	}
}

func TestCipherSuite(t *testing.T) {
	suite := &tls.CipherSuite{
		ID:                tls.TLS_AES_128_GCM_SHA256,
		Name:              "TLS_AES_128_GCM_SHA256",
		SupportedVersions: []uint16{tls.VersionTLS13},
		Insecure:          false,
	}

	if suite.ID != tls.TLS_AES_128_GCM_SHA256 {
		t.Errorf("CipherSuite.ID = %#x, want %#x", suite.ID, tls.TLS_AES_128_GCM_SHA256)
	}
	if suite.Name != "TLS_AES_128_GCM_SHA256" {
		t.Errorf("CipherSuite.Name = %q, want TLS_AES_128_GCM_SHA256", suite.Name)
	}
}

func TestConstants(t *testing.T) {
	if tls.VersionSSL30 != 0x0300 {
		t.Errorf("VersionSSL30 = %#x, want 0x0300", tls.VersionSSL30)
	}
	if tls.VersionTLS10 != 0x0301 {
		t.Errorf("VersionTLS10 = %#x, want 0x0301", tls.VersionTLS10)
	}
	if tls.VersionTLS11 != 0x0302 {
		t.Errorf("VersionTLS11 = %#x, want 0x0302", tls.VersionTLS11)
	}
	if tls.VersionTLS12 != 0x0303 {
		t.Errorf("VersionTLS12 = %#x, want 0x0303", tls.VersionTLS12)
	}
	if tls.VersionTLS13 != 0x0304 {
		t.Errorf("VersionTLS13 = %#x, want 0x0304", tls.VersionTLS13)
	}
}

func TestClientServer(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	serverConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()

	tlsServer := tls.Server(serverConn, serverConfig)
	if tlsServer == nil {
		t.Error("tls.Server returned nil")
	}

	tlsClient := tls.Client(clientConn, &tls.Config{
		InsecureSkipVerify: true,
	})
	if tlsClient == nil {
		t.Error("tls.Client returned nil")
	}

	if err := tlsServer.Close(); err != nil {
		t.Fatalf("tlsServer.Close: %v", err)
	}
	if err := tlsClient.Close(); err != nil {
		t.Fatalf("tlsClient.Close: %v", err)
	}
}

func TestConnMethods(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	tlsConn := tls.Server(serverConn, config)

	state := tlsConn.ConnectionState()
	if state.Version == 0 {
		t.Log("ConnectionState before handshake has zero version (expected)")
	}

	if err := tlsConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("Conn.SetDeadline: %v", err)
	}
	if err := tlsConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("Conn.SetReadDeadline: %v", err)
	}
	if err := tlsConn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("Conn.SetWriteDeadline: %v", err)
	}

	if err := tlsConn.Close(); err != nil {
		t.Fatalf("tlsConn.Close: %v", err)
	}
	if err := clientConn.Close(); err != nil {
		t.Fatalf("clientConn.Close: %v", err)
	}
}

func TestEncryptedClientHelloKey(t *testing.T) {
	key := &tls.EncryptedClientHelloKey{
		Config:      []byte{0x01, 0x02, 0x03},
		PrivateKey:  []byte{0x04, 0x05},
		SendAsRetry: true,
	}

	if len(key.Config) != 3 {
		t.Errorf("EncryptedClientHelloKey.Config length = %d, want 3", len(key.Config))
	}
	if !key.SendAsRetry {
		t.Error("EncryptedClientHelloKey.SendAsRetry = false, want true")
	}
}

func TestConfigBuildNameToCertificate(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	config.BuildNameToCertificate()
}

func TestClientAuthTypeString(t *testing.T) {
	tests := []tls.ClientAuthType{
		tls.NoClientCert,
		tls.RequestClientCert,
		tls.RequireAnyClientCert,
		tls.VerifyClientCertIfGiven,
		tls.RequireAndVerifyClientCert,
	}
	for _, authType := range tests {
		str := authType.String()
		if str == "" {
			t.Errorf("ClientAuthType(%d).String() returned empty string", authType)
		}
	}
}

func TestCurveIDString(t *testing.T) {
	curves := []tls.CurveID{
		tls.CurveP256,
		tls.CurveP384,
		tls.CurveP521,
		tls.X25519,
	}
	for _, curve := range curves {
		str := curve.String()
		if str == "" {
			t.Errorf("CurveID(%d).String() returned empty string", curve)
		}
	}
}

func TestSignatureSchemeString(t *testing.T) {
	schemes := []tls.SignatureScheme{
		tls.PKCS1WithSHA256,
		tls.PSSWithSHA256,
		tls.ECDSAWithP256AndSHA256,
	}
	for _, scheme := range schemes {
		str := scheme.String()
		if str == "" {
			t.Errorf("SignatureScheme(%d).String() returned empty string", scheme)
		}
	}
}

func TestQUICEncryptionLevelString(t *testing.T) {
	levels := []tls.QUICEncryptionLevel{
		tls.QUICEncryptionLevelInitial,
		tls.QUICEncryptionLevelEarly,
		tls.QUICEncryptionLevelHandshake,
		tls.QUICEncryptionLevelApplication,
	}
	for _, level := range levels {
		str := level.String()
		if str == "" {
			t.Errorf("QUICEncryptionLevel(%d).String() returned empty string", level)
		}
	}
}

func TestCipherSuiteConstants(t *testing.T) {
	constants := []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_FALLBACK_SCSV,
	}
	for _, constant := range constants {
		if constant == 0 {
			t.Error("Cipher suite constant is zero")
		}
	}
}

func TestConnMethods2(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	tlsConn := tls.Server(serverConn, config)
	defer tlsConn.Close()

	localAddr := tlsConn.LocalAddr()
	if localAddr == nil {
		t.Error("Conn.LocalAddr() returned nil")
	}

	remoteAddr := tlsConn.RemoteAddr()
	if remoteAddr == nil {
		t.Error("Conn.RemoteAddr() returned nil")
	}

	netConn := tlsConn.NetConn()
	if netConn == nil {
		t.Error("Conn.NetConn() returned nil")
	}

	ocspResponse := tlsConn.OCSPResponse()
	if ocspResponse == nil {
		t.Log("Conn.OCSPResponse() returned nil (expected before handshake)")
	}

	err = tlsConn.VerifyHostname("example.com")
	if err == nil {
		t.Log("Conn.VerifyHostname succeeded (unexpected)")
	}
}

func TestQUICConnMethods(t *testing.T) {
	config := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			ServerName: "example.com",
			MinVersion: tls.VersionTLS13,
		},
	}
	conn := tls.QUICClient(config)
	defer conn.Close()

	err := conn.Start(context.Background())
	if err != nil {
		t.Logf("QUICConn.Start error: %v", err)
	}

	err = conn.HandleData(tls.QUICEncryptionLevelInitial, []byte{})
	if err != nil {
		t.Logf("QUICConn.HandleData error: %v", err)
	}

	conn.SetTransportParameters([]byte{})

	state := conn.ConnectionState()
	if state.Version == 0 {
		t.Log("QUICConn.ConnectionState before handshake has zero version")
	}
}

func TestQUICServerMethods(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.QUICConfig{
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		},
	}
	conn := tls.QUICServer(config)
	defer conn.Close()

	err = conn.SendSessionTicket(tls.QUICSessionTicketOptions{EarlyData: true})
	if err != nil {
		t.Logf("QUICConn.SendSessionTicket error: %v", err)
	}

	err = conn.StoreSession(nil)
	if err != nil {
		t.Logf("QUICConn.StoreSession error: %v", err)
	}
}

func TestClientSessionCache(t *testing.T) {
	var cache tls.ClientSessionCache
	cache = tls.NewLRUClientSessionCache(10)
	if cache == nil {
		t.Error("NewLRUClientSessionCache returned nil")
	}
}

func TestClientSessionStateResumptionState(t *testing.T) {
	sessionState := &tls.SessionState{}
	ticket := []byte("test-ticket")
	clientState, err := tls.NewResumptionState(ticket, sessionState)
	if err != nil {
		t.Logf("NewResumptionState error: %v", err)
		return
	}

	ticket2, resumptionState, err := clientState.ResumptionState()
	if err != nil {
		t.Logf("ClientSessionState.ResumptionState error: %v", err)
	}
	if ticket2 == nil || resumptionState == nil {
		t.Error("ClientSessionState.ResumptionState returned nil")
	}
}

func TestCertificateRequestInfoContext(t *testing.T) {
	info := &tls.CertificateRequestInfo{
		AcceptableCAs: [][]byte{{0x01, 0x02}},
		Version:       tls.VersionTLS13,
	}

	if info.Context() != nil {
		t.Fatal("CertificateRequestInfo.Context() should be nil without handshake context")
	}
}

func TestClientHelloInfoContext(t *testing.T) {
	info := &tls.ClientHelloInfo{
		ServerName: "example.com",
	}

	if info.Context() != nil {
		t.Fatal("ClientHelloInfo.Context() should be nil without handshake context")
	}
}

func TestConfigEncryptDecryptTicket(t *testing.T) {
	config := &tls.Config{}

	state := tls.ConnectionState{Version: tls.VersionTLS13}
	sessionState := &tls.SessionState{}

	encrypted, err := config.EncryptTicket(state, sessionState)
	if err != nil {
		t.Logf("Config.EncryptTicket error: %v (expected without setup)", err)
	}
	if encrypted != nil {
		t.Logf("Config.EncryptTicket returned %d bytes", len(encrypted))
	}

	decrypted, err := config.DecryptTicket([]byte("test"), state)
	if err != nil {
		t.Logf("Config.DecryptTicket error: %v (expected without setup)", err)
	}
	if decrypted != nil {
		t.Log("Config.DecryptTicket returned session state")
	}
}

func TestDialerDialMethods(t *testing.T) {
	dialer := &tls.Dialer{
		Config: &tls.Config{
			ServerName:         "example.com",
			InsecureSkipVerify: true,
		},
	}

	_, err := dialer.Dial("tcp", "invalid-address:443")
	if err == nil {
		t.Error("Dialer.Dial should fail with invalid address")
	}

	_, err = dialer.DialContext(context.Background(), "tcp", "invalid-address:443")
	if err == nil {
		t.Error("Dialer.DialContext should fail with invalid address")
	}
}

func TestListenAndNewListener(t *testing.T) {
	ln, err := tls.Listen("tcp", "invalid-address:443", nil)
	if err == nil {
		ln.Close()
		t.Error("tls.Listen should fail with invalid address")
	}

	testListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer testListener.Close()

	tlsListener := tls.NewListener(testListener, &tls.Config{})
	if tlsListener == nil {
		t.Error("tls.NewListener returned nil")
	}
	tlsListener.Close()
}

func TestConnReadWrite(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	tlsConn := tls.Server(serverConn, config)
	defer tlsConn.Close()

	buf := make([]byte, 10)
	if err := tlsConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("Conn.SetReadDeadline: %v", err)
	}
	_, err = tlsConn.Read(buf)
	if err != nil {
		t.Logf("Conn.Read error: %v (expected before handshake)", err)
	}

	if err := tlsConn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("Conn.SetWriteDeadline: %v", err)
	}
	_, err = tlsConn.Write([]byte("test"))
	if err != nil {
		t.Logf("Conn.Write error: %v (expected before handshake)", err)
	}
}

func TestConnHandshake(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	tlsConn := tls.Server(serverConn, config)
	defer tlsConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tlsConn.HandshakeContext(ctx)
	if err != nil {
		t.Logf("Conn.Handshake error: %v (expected without client)", err)
	}
}

func TestConnHandshakeTimeout(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	tlsConn := tls.Server(serverConn, config)
	defer tlsConn.Close()

	if err := tlsConn.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("Conn.SetDeadline: %v", err)
	}
	if err := tlsConn.Handshake(); err == nil {
		t.Error("Conn.Handshake should fail without client peer")
	}
}

func TestConnCloseWrite(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	tlsConn := tls.Server(serverConn, config)
	defer tlsConn.Close()

	err = tlsConn.CloseWrite()
	if err != nil {
		t.Logf("Conn.CloseWrite error: %v", err)
	}
}

func TestConnectionStateExportKeyingMaterial(t *testing.T) {
	state := tls.ConnectionState{
		Version:           tls.VersionTLS13,
		HandshakeComplete: true,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic due to missing exporter state")
		}
	}()
	state.ExportKeyingMaterial("label", nil, 32)
}

func TestConnHandshakeContext(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair failed: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	tlsConn := tls.Server(serverConn, config)
	defer tlsConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tlsConn.HandshakeContext(ctx)
	if err == nil {
		t.Error("Conn.HandshakeContext should fail without peer input")
	}
}
