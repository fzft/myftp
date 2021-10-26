package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Command interface {
	Execute(*FtpConn, string)
	ParamsRequired() bool
	AuthRequired() bool
}

type CommandMap map[string]Command

var (
	commands = CommandMap{
		"USER": CommandUser{},
		"PASS": CommandPass{},
		"QUIT": CommandQuit{},
		"TYPE": CommandType{},
		"EPSV": CommandEPSV{},
		"PASV": CommandPASV{},
		"STOR": CommandSTOR{},
	}
)

type CommandUser struct {
}

func (cmd CommandUser) ParamsRequired() bool {
	return true
}
func (cmd CommandUser) AuthRequired() bool {
	return false
}

func (cmd CommandUser) Execute(conn *FtpConn, param string) {
	if conn.auth.need {
		if conn.auth.username == param {
			// username validate ok
			conn.WriteMessage(StatusUsernameOK, "User name okay, need password.")
			return
		} else {
			// username invalid
			conn.WriteMessage(StatusInvalidUsernameOrPass, "Invalid username or password")
			return
		}
	} else {
		conn.WriteMessage(StatusActionSuccessfully, "The requested action has been successfully completed.")
	}
}

type CommandPass struct {
}

func (cmd CommandPass) ParamsRequired() bool {
	return false
}
func (cmd CommandPass) AuthRequired() bool {
	return false
}

func (cmd CommandPass) Execute(conn *FtpConn, param string) {
	if conn.auth.need {
		if conn.auth.password == param {
			conn.WriteMessage(StatusUserLogin, "User logged in")
			conn.isLogin = true
		} else {
			conn.WriteMessage(StatusInvalidUsernameOrPass, "Invalid username or password")
		}
	} else {

	}
}

type CommandQuit struct {
}

func (cmd CommandQuit) ParamsRequired() bool {
	return true
}
func (cmd CommandQuit) AuthRequired() bool {
	return true
}

func (cmd CommandQuit) Execute(conn *FtpConn, param string) {
	conn.WriteMessage(StatusServiceClosing, "bye")
	conn.Close()
}

type CommandType struct {
}

func (cmd CommandType) ParamsRequired() bool {
	return false
}
func (cmd CommandType) AuthRequired() bool {
	return false
}

func (cmd CommandType) Execute(conn *FtpConn, param string) {
	if strings.ToUpper(param) == "A" {
		conn.WriteMessage(StatusActionSuccessfully, "Type set to ASCII")
	} else if strings.ToUpper(param) == "I" {
		conn.WriteMessage(StatusActionSuccessfully, "Type set to binary")
	} else {
	}
}

type CommandEPSV struct {
}

func (cmd CommandEPSV) ParamsRequired() bool {
	return false
}
func (cmd CommandEPSV) AuthRequired() bool {
	return true
}

func (cmd CommandEPSV) Execute(conn *FtpConn, param string) {
	lnIp := conn.passiveListenIP()
	socket, err := NewFtpPassiveSocket(lnIp)
	if err != nil {
		fmt.Println("err", err)
		conn.WriteMessage(StatusCannotOpenDataConnection, "Can't open data connection.")
		return
	}
	conn.dataConn = socket
	target := fmt.Sprintf("(%d)", socket.Port())
	msg := "Entering Extended Passive Mode " + target
	conn.logger.Infof("epsv response: %s", msg)
	conn.WriteMessage(StatusEnterExtendedPassiveMode, msg)
}

type CommandPASV struct {
}

func (cmd CommandPASV) ParamsRequired() bool {
	return false
}
func (cmd CommandPASV) AuthRequired() bool {
	return true
}

func (cmd CommandPASV) Execute(conn *FtpConn, param string) {
	lnIp := conn.passiveListenIP()
	socket, err := NewFtpPassiveSocket(lnIp)
	if err != nil {
		conn.WriteMessage(StatusCannotOpenDataConnection, "Can't open data connection.")
		return
	}
	conn.dataConn = socket
	p1 := socket.Port() / 256
	p2 := socket.Port() - (p1 * 256)
	quads := strings.Split(lnIp, ".")
	target := fmt.Sprintf("(%s,%s,%s,%s,%d,%d)", quads[0], quads[1], quads[2], quads[3], p1, p2)
	msg := "Entering Passive Mode " + target
	conn.logger.Infof("pasv response: %s", msg)
	conn.WriteMessage(StatusEnterPassiveMode, msg)
}

type CommandSTOR struct {
}

func (cmd CommandSTOR) ParamsRequired() bool {
	return false
}
func (cmd CommandSTOR) AuthRequired() bool {
	return true
}

func (cmd CommandSTOR) Execute(conn *FtpConn, param string) {

	targetPath := conn.buildPath(param)
	conn.WriteMessage(statusFileStatusOk, "File status okay; about to open data connection.")
	defer func() {
		conn.appendData = false
	}()

	bytes, err := conn.driver.PutFile(targetPath, conn.dataConn, conn.appendData)
	if err == nil {
		msg := "OK, received " + strconv.Itoa(int(bytes)) + " bytes"
		conn.logger.Infof("Stor msg: %s", msg)
		conn.WriteMessage(StatusFileActionSuccessful, msg)
	} else {
		conn.WriteMessage(StatusFileActionNotTaken, fmt.Sprint("error during transfer: ", err))
	}
}

