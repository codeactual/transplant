module origin.tld/user/proj

go 1.12

replace github.com/pmezard/go-difflib => github.com/pmezard/go-difflib v0.0.0-20151027124105-ac475e89e25c

replace github.com/golang/protobuf => github.com/golang/protobuf v1.0.0

require (
	github.com/mitchellh/mapstructure v1.1.0
	github.com/onsi/gomega v1.4.3
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f
)

replace github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.3.0

replace github.com/fsnotify/fsnotify => github.com/fsnotify/fsnotify v1.2.8
