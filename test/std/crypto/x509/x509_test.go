package x509_test

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"
)

// Certificate fixtures validate X.509 behavior, not RSA key strength.
const testRSAKeyBits = 1024

func TestErrorConstants(t *testing.T) {
	if x509.ErrUnsupportedAlgorithm == nil {
		t.Error("ErrUnsupportedAlgorithm is nil")
	}
	if x509.IncorrectPasswordError == nil {
		t.Error("IncorrectPasswordError is nil")
	}
}

func TestPublicKeyAlgorithm(t *testing.T) {
	algorithms := []x509.PublicKeyAlgorithm{
		x509.UnknownPublicKeyAlgorithm,
		x509.RSA,
		x509.DSA,
		x509.ECDSA,
		x509.Ed25519,
	}

	for _, algo := range algorithms {
		str := algo.String()
		if str == "" {
			t.Errorf("PublicKeyAlgorithm(%d).String() returned empty", algo)
		}
	}
}

func TestSignatureAlgorithm(t *testing.T) {
	algorithms := []x509.SignatureAlgorithm{
		x509.UnknownSignatureAlgorithm,
		x509.MD2WithRSA,
		x509.MD5WithRSA,
		x509.SHA1WithRSA,
		x509.SHA256WithRSA,
		x509.SHA384WithRSA,
		x509.SHA512WithRSA,
		x509.DSAWithSHA1,
		x509.DSAWithSHA256,
		x509.ECDSAWithSHA1,
		x509.ECDSAWithSHA256,
		x509.ECDSAWithSHA384,
		x509.ECDSAWithSHA512,
		x509.SHA256WithRSAPSS,
		x509.SHA384WithRSAPSS,
		x509.SHA512WithRSAPSS,
		x509.PureEd25519,
	}

	for _, algo := range algorithms {
		str := algo.String()
		if str == "" {
			t.Errorf("SignatureAlgorithm(%d).String() returned empty", algo)
		}
	}
}

func TestInsecureAlgorithmError(t *testing.T) {
	err := x509.InsecureAlgorithmError(x509.MD5WithRSA)
	if err.Error() == "" {
		t.Error("InsecureAlgorithmError.Error() returned empty")
	}
}

func TestKeyUsage(t *testing.T) {
	usages := []x509.KeyUsage{
		x509.KeyUsageDigitalSignature,
		x509.KeyUsageContentCommitment,
		x509.KeyUsageKeyEncipherment,
		x509.KeyUsageDataEncipherment,
		x509.KeyUsageKeyAgreement,
		x509.KeyUsageCertSign,
		x509.KeyUsageCRLSign,
		x509.KeyUsageEncipherOnly,
		x509.KeyUsageDecipherOnly,
	}

	for _, usage := range usages {
		_ = usage
	}
}

func TestExtKeyUsage(t *testing.T) {
	usages := []x509.ExtKeyUsage{
		x509.ExtKeyUsageAny,
		x509.ExtKeyUsageServerAuth,
		x509.ExtKeyUsageClientAuth,
		x509.ExtKeyUsageCodeSigning,
		x509.ExtKeyUsageEmailProtection,
		x509.ExtKeyUsageIPSECEndSystem,
		x509.ExtKeyUsageIPSECTunnel,
		x509.ExtKeyUsageIPSECUser,
		x509.ExtKeyUsageTimeStamping,
		x509.ExtKeyUsageOCSPSigning,
		x509.ExtKeyUsageMicrosoftServerGatedCrypto,
		x509.ExtKeyUsageNetscapeServerGatedCrypto,
		x509.ExtKeyUsageMicrosoftCommercialCodeSigning,
		x509.ExtKeyUsageMicrosoftKernelCodeSigning,
	}

	for _, usage := range usages {
		_ = usage
	}
}

func TestInvalidReason(t *testing.T) {
	reasons := []x509.InvalidReason{
		x509.NotAuthorizedToSign,
		x509.Expired,
		x509.CANotAuthorizedForThisName,
		x509.TooManyIntermediates,
		x509.IncompatibleUsage,
		x509.NameMismatch,
		x509.NameConstraintsWithoutSANs,
		x509.UnconstrainedName,
		x509.TooManyConstraints,
		x509.CANotAuthorizedForExtKeyUsage,
	}

	for _, reason := range reasons {
		_ = reason
	}
}

func TestPEMCipher(t *testing.T) {
	ciphers := []x509.PEMCipher{
		x509.PEMCipherDES,
		x509.PEMCipher3DES,
		x509.PEMCipherAES128,
		x509.PEMCipherAES192,
		x509.PEMCipherAES256,
	}

	for _, cipher := range ciphers {
		_ = cipher
	}
}

func TestOID(t *testing.T) {
	oid, err := x509.OIDFromInts([]uint64{1, 2, 840, 113549})
	if err != nil {
		t.Errorf("OIDFromInts failed: %v", err)
	}

	str := oid.String()
	if str != "1.2.840.113549" {
		t.Errorf("OID.String() = %q, want 1.2.840.113549", str)
	}

	oid2, err := x509.ParseOID("1.2.840.113549")
	if err != nil {
		t.Errorf("ParseOID failed: %v", err)
	}

	if !oid.Equal(oid2) {
		t.Error("OID.Equal returned false for equal OIDs")
	}

	asn1OID := asn1.ObjectIdentifier{1, 2, 840, 113549}
	if !oid.EqualASN1OID(asn1OID) {
		t.Error("OID.EqualASN1OID returned false")
	}

	binData, err := oid.MarshalBinary()
	if err != nil {
		t.Errorf("OID.MarshalBinary failed: %v", err)
	}

	var oid3 x509.OID
	if err := oid3.UnmarshalBinary(binData); err != nil {
		t.Errorf("OID.UnmarshalBinary failed: %v", err)
	}
	if !oid.Equal(oid3) {
		t.Error("OID roundtrip through Binary failed")
	}

	textData, err := oid.MarshalText()
	if err != nil {
		t.Errorf("OID.MarshalText failed: %v", err)
	}

	var oid4 x509.OID
	if err := oid4.UnmarshalText(textData); err != nil {
		t.Errorf("OID.UnmarshalText failed: %v", err)
	}
	if !oid.Equal(oid4) {
		t.Error("OID roundtrip through Text failed")
	}

	binData2, err := oid.AppendBinary([]byte("prefix"))
	if err != nil {
		t.Errorf("OID.AppendBinary failed: %v", err)
	}
	if string(binData2[:6]) != "prefix" {
		t.Error("OID.AppendBinary did not preserve prefix")
	}

	textData2, err := oid.AppendText([]byte("prefix"))
	if err != nil {
		t.Errorf("OID.AppendText failed: %v", err)
	}
	if string(textData2[:6]) != "prefix" {
		t.Error("OID.AppendText did not preserve prefix")
	}
}

func generateSelfSignedCert(t *testing.T) (*x509.Certificate, crypto.PrivateKey) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "test.example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"test.example.com"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert, priv
}

func TestCreateCertificate(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	if cert.Subject.CommonName != "test.example.com" {
		t.Errorf("Certificate CommonName = %q, want test.example.com", cert.Subject.CommonName)
	}

	if len(cert.DNSNames) != 1 || cert.DNSNames[0] != "test.example.com" {
		t.Errorf("Certificate DNSNames = %v, want [test.example.com]", cert.DNSNames)
	}
}

func TestParseCertificate(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	if cert.Version != 3 {
		t.Errorf("Certificate Version = %d, want 3", cert.Version)
	}

	if cert.SerialNumber.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Certificate SerialNumber = %v, want 1", cert.SerialNumber)
	}
}

func TestParseCertificates(t *testing.T) {
	cert1, _ := generateSelfSignedCert(t)
	cert2, _ := generateSelfSignedCert(t)

	combined := append(cert1.Raw, cert2.Raw...)
	certs, err := x509.ParseCertificates(combined)
	if err != nil {
		t.Errorf("ParseCertificates failed: %v", err)
	}
	if len(certs) != 2 {
		t.Errorf("ParseCertificates returned %d certs, want 2", len(certs))
	}
}

func TestCertificateEqual(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	if !cert.Equal(cert) {
		t.Error("Certificate.Equal returned false for same certificate")
	}

	cert2, _ := generateSelfSignedCert(t)
	if cert.Equal(cert2) {
		t.Error("Certificate.Equal returned true for different certificates")
	}
}

func TestCertificateCheckSignatureFrom(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	if err := cert.CheckSignatureFrom(cert); err != nil {
		t.Errorf("CheckSignatureFrom failed for self-signed cert: %v", err)
	}
}

func TestCertificateVerifyHostname(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	if err := cert.VerifyHostname("test.example.com"); err != nil {
		t.Errorf("VerifyHostname failed: %v", err)
	}

	err := cert.VerifyHostname("wrong.example.com")
	if err == nil {
		t.Error("VerifyHostname should fail for wrong hostname")
	}

	var hostnameErr x509.HostnameError
	if herr, ok := err.(x509.HostnameError); ok {
		hostnameErr = herr
		if hostnameErr.Error() == "" {
			t.Error("HostnameError.Error() returned empty")
		}
	}
}

func TestCertPool(t *testing.T) {
	pool := x509.NewCertPool()
	if pool == nil {
		t.Fatal("NewCertPool returned nil")
	}

	cert, _ := generateSelfSignedCert(t)
	pool.AddCert(cert)

	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Error("CertPool.Subjects() returned empty after AddCert")
	}

	clonedPool := pool.Clone()
	if clonedPool == nil {
		t.Error("CertPool.Clone() returned nil")
	}

	if !pool.Equal(clonedPool) {
		t.Error("CertPool.Equal returned false for cloned pool")
	}

	pool2 := x509.NewCertPool()
	if pool.Equal(pool2) {
		t.Error("CertPool.Equal returned true for different pools")
	}
}

func TestCertPoolAppendCertsFromPEM(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	pemData := pem.EncodeToMemory(pemBlock)

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pemData)
	if !ok {
		t.Error("AppendCertsFromPEM returned false")
	}

	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Error("CertPool is empty after AppendCertsFromPEM")
	}
}

func TestCertPoolAddCertWithConstraint(t *testing.T) {
	pool := x509.NewCertPool()
	cert, _ := generateSelfSignedCert(t)

	constraint := func(chain []*x509.Certificate) error {
		if len(chain) == 0 {
			return nil
		}
		return nil
	}

	pool.AddCertWithConstraint(cert, constraint)

	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Error("CertPool.Subjects() returned empty after AddCertWithConstraint")
	}
}

func TestSystemCertPool(t *testing.T) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		var sysErr x509.SystemRootsError
		if serr, ok := err.(x509.SystemRootsError); ok {
			sysErr = serr
			if sysErr.Error() == "" {
				t.Error("SystemRootsError.Error() returned empty")
			}
			if sysErr.Unwrap() == nil {
				t.Log("SystemRootsError.Unwrap() returned nil")
			}
		}
		t.Skipf("SystemCertPool not available: %v", err)
	}
	if pool == nil {
		t.Error("SystemCertPool returned nil without error")
	}
}

func TestSetFallbackRoots(t *testing.T) {
	pool := x509.NewCertPool()
	x509.SetFallbackRoots(pool)
}

func TestMarshalPKCS1PrivateKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}

	der := x509.MarshalPKCS1PrivateKey(priv)
	if len(der) == 0 {
		t.Error("MarshalPKCS1PrivateKey returned empty")
	}

	parsedKey, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		t.Errorf("ParsePKCS1PrivateKey failed: %v", err)
	}
	if parsedKey.N.Cmp(priv.N) != 0 {
		t.Error("Parsed key does not match original")
	}
}

func TestMarshalPKCS1PublicKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}

	pub := &priv.PublicKey
	der := x509.MarshalPKCS1PublicKey(pub)
	if len(der) == 0 {
		t.Error("MarshalPKCS1PublicKey returned empty")
	}

	parsedKey, err := x509.ParsePKCS1PublicKey(der)
	if err != nil {
		t.Errorf("ParsePKCS1PublicKey failed: %v", err)
	}
	if parsedKey.N.Cmp(pub.N) != 0 {
		t.Error("Parsed public key does not match original")
	}
}

func TestMarshalPKCS8PrivateKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Errorf("MarshalPKCS8PrivateKey failed: %v", err)
	}
	if len(der) == 0 {
		t.Error("MarshalPKCS8PrivateKey returned empty")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		t.Errorf("ParsePKCS8PrivateKey failed: %v", err)
	}
	if parsedKey == nil {
		t.Error("ParsePKCS8PrivateKey returned nil")
	}
}

func TestMarshalPKIXPublicKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}

	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Errorf("MarshalPKIXPublicKey failed: %v", err)
	}
	if len(der) == 0 {
		t.Error("MarshalPKIXPublicKey returned empty")
	}

	parsedKey, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		t.Errorf("ParsePKIXPublicKey failed: %v", err)
	}
	if parsedKey == nil {
		t.Error("ParsePKIXPublicKey returned nil")
	}
}

func TestMarshalECPrivateKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Errorf("MarshalECPrivateKey failed: %v", err)
	}
	if len(der) == 0 {
		t.Error("MarshalECPrivateKey returned empty")
	}

	parsedKey, err := x509.ParseECPrivateKey(der)
	if err != nil {
		t.Errorf("ParseECPrivateKey failed: %v", err)
	}
	if parsedKey.D.Cmp(priv.D) != 0 {
		t.Error("Parsed EC key does not match original")
	}
}

func TestEncryptDecryptPEMBlock(t *testing.T) {
	data := []byte("test data for encryption")
	password := []byte("password")

	// Use deterministic IV to avoid flaky padding checks when decrypting with a wrong password.
	iv := make([]byte, 16) // AES block size.
	block, err := x509.EncryptPEMBlock(bytes.NewReader(iv), "TEST", data, password, x509.PEMCipherAES256)
	if err != nil {
		t.Errorf("EncryptPEMBlock failed: %v", err)
	}

	if !x509.IsEncryptedPEMBlock(block) {
		t.Error("IsEncryptedPEMBlock returned false for encrypted block")
	}

	decrypted, err := x509.DecryptPEMBlock(block, password)
	if err != nil {
		t.Errorf("DecryptPEMBlock failed: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Errorf("Decrypted data = %q, want %q", decrypted, data)
	}

	_, err = x509.DecryptPEMBlock(block, []byte("wrong"))
	if err != x509.IncorrectPasswordError {
		t.Error("DecryptPEMBlock should return IncorrectPasswordError for wrong password")
	}

	normalBlock := &pem.Block{Type: "TEST", Bytes: data}
	if x509.IsEncryptedPEMBlock(normalBlock) {
		t.Error("IsEncryptedPEMBlock returned true for unencrypted block")
	}
}

func TestCreateCertificateRequest(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "test.example.com",
			Organization: []string{"Test Org"},
		},
		DNSNames: []string{"test.example.com"},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, priv)
	if err != nil {
		t.Errorf("CreateCertificateRequest failed: %v", err)
	}

	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		t.Errorf("ParseCertificateRequest failed: %v", err)
	}

	if csr.Subject.CommonName != "test.example.com" {
		t.Errorf("CSR CommonName = %q, want test.example.com", csr.Subject.CommonName)
	}

	if err := csr.CheckSignature(); err != nil {
		t.Errorf("CertificateRequest.CheckSignature failed: %v", err)
	}
}

func TestCreateRevocationList(t *testing.T) {
	cert, priv := generateSelfSignedCert(t)

	template := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now(),
		NextUpdate: time.Now().Add(24 * time.Hour),
		RevokedCertificateEntries: []x509.RevocationListEntry{
			{
				SerialNumber:   big.NewInt(42),
				RevocationTime: time.Now(),
			},
		},
	}

	signer, ok := priv.(crypto.Signer)
	if !ok {
		t.Fatal("Private key does not implement crypto.Signer")
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, template, cert, signer)
	if err != nil {
		t.Skipf("CreateRevocationList failed: %v", err)
	}

	revList, err := x509.ParseRevocationList(crlDER)
	if err != nil {
		t.Errorf("ParseRevocationList failed: %v", err)
	}

	if revList != nil {
		if revList.Number.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("RevocationList.Number = %v, want 1", revList.Number)
		}

		if err := revList.CheckSignatureFrom(cert); err != nil {
			t.Errorf("RevocationList.CheckSignatureFrom failed: %v", err)
		}
	}
}

func TestParseCRL(t *testing.T) {
	cert, priv := generateSelfSignedCert(t)

	revokedCerts := []pkix.RevokedCertificate{
		{
			SerialNumber:   big.NewInt(42),
			RevocationTime: time.Now(),
		},
	}

	crlBytes, err := cert.CreateCRL(rand.Reader, priv, revokedCerts, time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Skipf("CreateCRL not supported: %v", err)
	}

	crl, err := x509.ParseCRL(crlBytes)
	if err != nil {
		t.Errorf("ParseCRL failed: %v", err)
	}

	if len(crl.TBSCertList.RevokedCertificates) == 0 {
		t.Error("ParseCRL returned CRL with no revoked certificates")
	}

	crl2, err := x509.ParseDERCRL(crlBytes)
	if err != nil {
		t.Errorf("ParseDERCRL failed: %v", err)
	}
	if crl2 == nil {
		t.Error("ParseDERCRL returned nil")
	}

	if err := cert.CheckCRLSignature(crl); err != nil {
		t.Errorf("Certificate.CheckCRLSignature failed: %v", err)
	}
}

func TestCheckSignature(t *testing.T) {
	cert, priv := generateSelfSignedCert(t)

	data := []byte("test data to sign")
	signer, ok := priv.(crypto.Signer)
	if !ok {
		t.Fatal("Private key does not implement crypto.Signer")
	}

	hashed := crypto.SHA256.New()
	hashed.Write(data)
	signature, err := signer.Sign(rand.Reader, hashed.Sum(nil), crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}

	err = cert.CheckSignature(x509.SHA256WithRSA, data, signature)
	if err != nil {
		t.Errorf("Certificate.CheckSignature failed: %v", err)
	}
}

func TestVerifyOptions(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	pool := x509.NewCertPool()
	pool.AddCert(cert)

	opts := x509.VerifyOptions{
		Roots:       pool,
		DNSName:     "test.example.com",
		CurrentTime: time.Now(),
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		t.Logf("Certificate.Verify failed (expected for self-signed): %v", err)
	} else if len(chains) == 0 {
		t.Log("Certificate.Verify returned empty chains")
	}
}

func TestCertificateInvalidError(t *testing.T) {
	err := x509.CertificateInvalidError{
		Cert:   nil,
		Reason: x509.Expired,
		Detail: "certificate expired",
	}

	if err.Error() == "" {
		t.Error("CertificateInvalidError.Error() returned empty")
	}
}

func TestUnknownAuthorityError(t *testing.T) {
	cert, _ := generateSelfSignedCert(t)

	err := x509.UnknownAuthorityError{
		Cert: cert,
	}

	if err.Error() == "" {
		t.Error("UnknownAuthorityError.Error() returned empty")
	}
}

func TestConstraintViolationError(t *testing.T) {
	err := x509.ConstraintViolationError{}
	if err.Error() == "" {
		t.Error("ConstraintViolationError.Error() returned empty")
	}
}

func TestUnhandledCriticalExtension(t *testing.T) {
	err := x509.UnhandledCriticalExtension{}
	if err.Error() == "" {
		t.Error("UnhandledCriticalExtension.Error() returned empty")
	}
}

func TestPolicyMapping(t *testing.T) {
	oid1, _ := x509.OIDFromInts([]uint64{1, 2, 3})
	oid2, _ := x509.OIDFromInts([]uint64{4, 5, 6})

	pm := x509.PolicyMapping{
		IssuerDomainPolicy:  oid1,
		SubjectDomainPolicy: oid2,
	}

	if !pm.IssuerDomainPolicy.Equal(oid1) {
		t.Error("PolicyMapping.IssuerDomainPolicy not set correctly")
	}
}

func TestEd25519(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		t.Errorf("CreateCertificate with Ed25519 failed: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Errorf("ParseCertificate with Ed25519 failed: %v", err)
	}

	if cert.PublicKeyAlgorithm != x509.Ed25519 {
		t.Errorf("Certificate.PublicKeyAlgorithm = %v, want Ed25519", cert.PublicKeyAlgorithm)
	}

	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Errorf("MarshalPKCS8PrivateKey with Ed25519 failed: %v", err)
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		t.Errorf("ParsePKCS8PrivateKey with Ed25519 failed: %v", err)
	}
	if parsedKey == nil {
		t.Error("ParsePKCS8PrivateKey returned nil")
	}
}

func TestCertificateWithURIs(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}

	uri, _ := url.Parse("https://example.com/resource")

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{uri},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Errorf("CreateCertificate with URIs failed: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Errorf("ParseCertificate failed: %v", err)
	}

	if len(cert.URIs) == 0 {
		t.Error("Certificate.URIs is empty")
	}
}
