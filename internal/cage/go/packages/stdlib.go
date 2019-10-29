// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"path"
	"path/filepath"
	"strings"

	cage_build "github.com/codeactual/transplant/internal/cage/go/build"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// stdlibImportPaths allows us to reduce go/build use for Goroot=true checks.
//
// It is generated with this command from $GOROOT/src: go list ./... | egrep -v "^(internal|cmd)"
var stdlibImportPaths *cage_strings.Set

// stdlibPackageNames is derived from stdlibImportPaths.
var stdlibPackageNames *cage_strings.Set

func init() {
	stdlibImportPaths = cage_strings.NewSet().AddSlice(
		[]string{
			"archive/tar",
			"archive/zip",
			"bufio",
			"bytes",
			"builtin", // added manually, "go list" no longer includes it (last check: go1.13.3)
			"compress/bzip2",
			"compress/flate",
			"compress/gzip",
			"compress/lzw",
			"compress/zlib",
			"container/heap",
			"container/list",
			"container/ring",
			"context",
			"crypto",
			"crypto/aes",
			"crypto/cipher",
			"crypto/des",
			"crypto/dsa",
			"crypto/ecdsa",
			"crypto/ed25519",
			"crypto/ed25519/internal/edwards25519",
			"crypto/elliptic",
			"crypto/hmac",
			"crypto/internal/randutil",
			"crypto/internal/subtle",
			"crypto/md5",
			"crypto/rand",
			"crypto/rc4",
			"crypto/rsa",
			"crypto/sha1",
			"crypto/sha256",
			"crypto/sha512",
			"crypto/subtle",
			"crypto/tls",
			"crypto/x509",
			"crypto/x509/pkix",
			"database/sql",
			"database/sql/driver",
			"debug/dwarf",
			"debug/elf",
			"debug/gosym",
			"debug/macho",
			"debug/pe",
			"debug/plan9obj",
			"encoding",
			"encoding/ascii85",
			"encoding/asn1",
			"encoding/base32",
			"encoding/base64",
			"encoding/binary",
			"encoding/csv",
			"encoding/gob",
			"encoding/hex",
			"encoding/json",
			"encoding/pem",
			"encoding/xml",
			"errors",
			"expvar",
			"flag",
			"fmt",
			"go/ast",
			"go/build",
			"go/constant",
			"go/doc",
			"go/format",
			"go/importer",
			"go/internal/gccgoimporter",
			"go/internal/gcimporter",
			"go/internal/srcimporter",
			"go/parser",
			"go/printer",
			"go/scanner",
			"go/token",
			"go/types",
			"hash",
			"hash/adler32",
			"hash/crc32",
			"hash/crc64",
			"hash/fnv",
			"html",
			"html/template",
			"image",
			"image/color",
			"image/color/palette",
			"image/draw",
			"image/gif",
			"image/internal/imageutil",
			"image/jpeg",
			"image/png",
			"index/suffixarray",
			"io",
			"io/ioutil",
			"log",
			"log/syslog",
			"math",
			"math/big",
			"math/bits",
			"math/cmplx",
			"math/rand",
			"mime",
			"mime/multipart",
			"mime/quotedprintable",
			"net",
			"net/http",
			"net/http/cgi",
			"net/http/cookiejar",
			"net/http/fcgi",
			"net/http/httptest",
			"net/http/httptrace",
			"net/http/httputil",
			"net/http/internal",
			"net/http/pprof",
			"net/internal/socktest",
			"net/mail",
			"net/rpc",
			"net/rpc/jsonrpc",
			"net/smtp",
			"net/textproto",
			"net/url",
			"os",
			"os/exec",
			"os/signal",
			"os/user",
			"path",
			"path/filepath",
			"plugin",
			"reflect",
			"regexp",
			"regexp/syntax",
			"runtime",
			"runtime/cgo",
			"runtime/debug",
			"runtime/internal/atomic",
			"runtime/internal/math",
			"runtime/internal/sys",
			"runtime/pprof",
			"runtime/pprof/internal/profile",
			"runtime/race",
			"runtime/trace",
			"sort",
			"strconv",
			"strings",
			"sync",
			"sync/atomic",
			"syscall",
			"testing",
			"testing/internal/testdeps",
			"testing/iotest",
			"testing/quick",
			"text/scanner",
			"text/tabwriter",
			"text/template",
			"text/template/parse",
			"time",
			"unicode",
			"unicode/utf16",
			"unicode/utf8",
			"unsafe",
		})

	stdlibPackageNames = cage_strings.NewSet()
	for _, i := range stdlibImportPaths.Slice() {
		stdlibPackageNames.Add(path.Base(i))
	}
}

func StdlibDir(p string) string {
	return filepath.Join(append([]string{cage_build.Goroot(), "src"}, strings.Split(p, "/")...)...)
}
