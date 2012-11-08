/*
	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package main

import (
	//"github.com/xiam/hyperfox/director/inject"
	"github.com/xiam/hyperfox/logger"
	"github.com/xiam/hyperfox/proxy"
	"github.com/xiam/hyperfox/writer/save"
	"os"
)

func main() {
	p := proxy.New()

	p.AddDirector(logger.Client(os.Stdout))

	p.AddDirector(logger.Request)
	p.AddDirector(logger.Head)
	p.AddDirector(logger.Body)

	p.AddWriter(save.Head)
	p.AddWriter(save.Body)
	p.AddWriter(save.Response)

	p.AddLogger(logger.Server(os.Stdout))

	p.Start()
}
