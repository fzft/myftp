package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

type Action int

const (
	None Action = iota
)

type Event interface {
	OnData(c *FtpConn, in []byte) (out []byte, action Action)
}

type FtpEventHandler struct {
	logger *logrus.Logger
	auth   Auth
	isLogin bool
}

func (e *FtpEventHandler) OnData(c *FtpConn, in []byte) (out []byte, action Action) {
	line := fmt.Sprintf("%s", strings.TrimSpace(string(in)))
	out, _ = e.receiveLine(line, c)
	return out, None
}

func (e *FtpEventHandler) WriteMessage(code int, message string) ([]byte, error) {
	line := fmt.Sprintf("%d %s\r\n", code, message)
	return []byte(line), nil
}

func (e *FtpEventHandler) buildPath(filename string) string {
	fullPath := filepath.Clean("." + "/" + filename)
	return fullPath
}

// receiveLine handle concrete read
func (e *FtpEventHandler) receiveLine(line string, c *FtpConn) ([]byte, error) {
	command, params := e.parseLine(line)
	e.logger.Infof("command %s, params %s\n", command, params)
	commandObj := commands[command]
	if commandObj == nil {
		return e.WriteMessage(StatusUnrecognizedCommand, "command unrecognized")
	}
	// validate auth required
	if commandObj.AuthRequired() && c.isLogin == false {

		return e.WriteMessage(StatusNotLogin, "Not logged in.")
	}
	return commandObj.Execute(e, params)
}

func (e *FtpEventHandler) parseLine(line string) (string, string) {
	params := strings.SplitN(strings.Trim(line, "\r\n"), " ", 2)
	if len(params) == 1 {
		return params[0], ""
	}
	return params[0], params[1]
}
