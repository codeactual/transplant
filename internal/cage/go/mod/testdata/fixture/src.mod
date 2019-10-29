module domain.com/path/to/src

go 1.12

replace replace0-old-domain.com/user/proj => replace0-new-domain.com/user/proj

replace replace1-old-domain.com/user/proj => replace1-new-domain.com/user/proj replace1-version

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20180906201452-2aa6f33b730c
	github.com/gorilla/securecookie v1.1.1
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/julienschmidt/httprouter v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/segmentio/ksuid v1.0.2
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.3.1
	github.com/tjarratt/babble v0.0.0-20140317234543-2cf06e8d98b0
	golang.org/x/crypto v0.0.0-20190123085648-057139ce5d2b
)

replace replace2-old-domain.com/user/proj => replace2-new-domain.com/user/proj replace2-version

replace replace3-old-domain.com/user/proj => replace3-new-domain.com/user/proj
