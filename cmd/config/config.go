package config

import (
	"flag"
	"time"
)

var cliSrc = flag.String("clisrc", ":18080", `src addr`)
var cliDst = flag.String("clidst", "localhost:18081", `src addr`)
var svrSrc = flag.String("svrsrc", ":18081", `src addr`)
var svrDst = flag.String("svrdst", "localhost:18082", `src addr`)
var timeout = flag.Int("t", 120, `read timeout`)
var passwd = flag.String("p", "7yuhdjamfklsdfk$%^&*;d/,.cx,vzbn18276312ojskdlfjal;djfka;", `password`)

func init() {
	flag.Parse()
}

func KeyIV() (key, iv []byte) {
	pass := *passwd
	for len(pass) < 32 {
		pass += pass
	}

	key = []byte(pass)[:32]
	iv = key[8:24]
	return key, iv
}

func ClientAddrs() (string, string) {
	return *cliSrc, *cliDst
}

func ServerAddrs() (string, string) {
	return *svrSrc, *svrDst
}

func Timeout() time.Duration {
	return time.Second * time.Duration(*timeout)
}
