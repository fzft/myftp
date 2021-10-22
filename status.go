package main

// from https://en.wikipedia.org/wiki/List_of_FTP_server_return_codes
const (
	StatusActionSuccessfully = 200
	StatusReadyForNewUser    = 220
	StatusServiceClosing = 221
	StatusEnterPassiveMode = 227
	StatusUserLogin = 230

	StatusUsernameOK          = 331
	StatusNeedAccountForLogin = 332

	StatusCannotOpenDataConnection = 425
	StatusInvalidUsernameOrPass = 430

	StatusUnrecognizedCommand = 500
	StatusNotLogin            = 530
)
