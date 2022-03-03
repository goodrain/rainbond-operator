package commonutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

//CA ca
type CA struct {
	caInfo          *x509.Certificate
	caPrivKey       *rsa.PrivateKey
	caPem, caKeyPem []byte
}

//GetCAPem get ca pem bytes
func (c *CA) GetCAPem() ([]byte, error) {
	if c.caPem == nil {
		// create the CA
		caBytes, err := x509.CreateCertificate(rand.Reader, c.caInfo, c.caInfo, &c.caPrivKey.PublicKey, c.caPrivKey)
		if err != nil {
			return nil, err
		}
		// pem encode
		caPEM := new(bytes.Buffer)
		_ = pem.Encode(caPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caBytes,
		})
		c.caPem = caPEM.Bytes()
	}
	return c.caPem, nil
}

//GetCAKeyPem get ca key pem
func (c *CA) GetCAKeyPem() ([]byte, error) {
	if c.caKeyPem == nil {
		caPrivKeyPEM := new(bytes.Buffer)
		_ = pem.Encode(caPrivKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(c.caPrivKey),
		})
		c.caKeyPem = caPrivKeyPEM.Bytes()
	}
	return c.caKeyPem, nil
}

//CreateCert make Certificate
func (c *CA) CreateCert(ips []string, domains ...string) (certPem, certKey []byte, err error) {
	var ipAddresses []net.IP
	for _, ip := range ips {
		if i := net.ParseIP(ip); i != nil {
			ipAddresses = append(ipAddresses, i)
		}
	}
	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Wutong, INC."},
			Country:       []string{"CN"},
			Province:      []string{"Beijing"},
			Locality:      []string{"Beijing"},
			StreetAddress: []string{"Beijing"},
			PostalCode:    []string{"000000"},
		},
		DNSNames:     domains,
		IPAddresses:  ipAddresses,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(99, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, c.caInfo, &certPrivKey.PublicKey, c.caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := new(bytes.Buffer)
	_ = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	return certPEM.Bytes(), certPrivKeyPEM.Bytes(), nil
}

//CreateCA create ca info
func CreateCA() (*CA, error) {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Wutong, INC."},
			Country:       []string{"CN"},
			Province:      []string{"Beijing"},
			Locality:      []string{"Beijing"},
			StreetAddress: []string{"Beijing"},
			PostalCode:    []string{"000000"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(99, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	return &CA{
		caInfo:    ca,
		caPrivKey: caPrivKey,
	}, nil
}

//ParseCA parse caPem
func ParseCA(caPem, caKeyPem []byte) (*CA, error) {
	p := &pem.Block{}
	p, caPem = pem.Decode(caPem)
	ca, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, err
	}
	p2 := &pem.Block{}
	p2, caKeyPem = pem.Decode(caKeyPem)
	caKey, err := x509.ParsePKCS1PrivateKey(p2.Bytes)
	if err != nil {
		return nil, err
	}
	return &CA{
		caInfo:    ca,
		caPrivKey: caKey,
		caPem:     caPem,
		caKeyPem:  caKeyPem,
	}, nil
}

//DomainSign create cert
func DomainSign(ips []string, domains ...string) ([]byte, []byte, []byte, error) {
	ca, err := CreateCA()
	if err != nil {
		return nil, nil, nil, err
	}
	caPem, err := ca.GetCAPem()
	if err != nil {
		return nil, nil, nil, err
	}
	certPem, certKey, err := ca.CreateCert(ips, domains...)
	if err != nil {
		return nil, nil, nil, err
	}
	return caPem, certPem, certKey, nil
}
