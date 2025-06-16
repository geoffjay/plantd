module github.com/geoffjay/plantd/identity/pkg/client

go 1.24

require (
	github.com/geoffjay/plantd/core v0.0.0-20250608024831-6d6af927872f
	github.com/geoffjay/plantd/identity/internal v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
)

replace github.com/geoffjay/plantd/identity/internal => ../../internal 
