package main

// Config exists to demonstrate that this responsibility lies with the
// application implementing the ntrip.caster module and not the module itself
type Config struct {
	HTTP struct {
		Port string
	}
	HTTPS struct {
		Port            string
		CertificateFile string
		PrivateKeyFile  string
	}
	Cognito struct {
		UserPoolID string
		ClientID   string
	}
}
