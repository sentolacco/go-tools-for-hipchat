package tools

import (
	"strings"
	"github.com/jessevdk/go-flags"
	"net/url"
	"crypto/md5"
	"io"
	"encoding/hex"
)

var opts = struct {
	EncodeUrl struct {
		Part string `short:"p" long:"part" choice:"query" choice:"path" default:"query"`
		Args struct {
			Input []string `required:"1-1"`
		} `positional-args:"yes"`
	} `command:"encodeUrl"`
	HashMd5 struct {
		Args struct {
			Input []string `required:"1-1"`
		} `positional-args:"yes"`
	} `command:"hashMd5"`
}{}

func Tools(message string)(result string, err error) {
	args := strings.Split(message, " ")
	args = append(args[:0], args[1:]...)
	p := flags.NewNamedParser("/tools", flags.Default)
	p.AddGroup("Application Options", "", &opts)
	_, err = p.ParseArgs(args)
	if err != nil {
		return "", err
	}

	cmd := p.Active.Name
	arg := ""
	result = ""

	switch p.Active.Name {
	case "encodeUrl":
		arg = opts.EncodeUrl.Args.Input[0]
		switch opts.EncodeUrl.Part {
		case "path":
			result = url.PathEscape(arg)
		case "query":
			result = url.QueryEscape(arg)
		}
	case "hashMd5":
		arg = opts.HashMd5.Args.Input[0]
		h := md5.New()
		io.WriteString(h, arg)
		result = hex.EncodeToString(h.Sum(nil)[:])
	}

	return  cmd + " " + arg + " = " + result, nil
}
