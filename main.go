/*
	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package main

import (
	"github.com/xiam/hyperfox/director/inject"
	"github.com/xiam/hyperfox/logger"
	"github.com/xiam/hyperfox/proxy"
	"github.com/xiam/hyperfox/writer/save"
	"os"
)

func main() {
	p := proxy.New()

	p.AddDirector(inject.Body)
	p.AddDirector(inject.Head)

	p.AddLogger(logger.Simple(os.Stdout))

	p.AddWriter(save.Body)
	p.AddWriter(save.Head)

	p.Start()
}
