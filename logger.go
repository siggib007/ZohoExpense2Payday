package main

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	iVerbose  int
	objLogOut *os.File
}

func NewLogger(strLogFile string, iVerbose int) (*Logger, error) {
	objLogOut, err := os.OpenFile(strLogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		iVerbose:  iVerbose,
		objLogOut: objLogOut,
	}, nil
}

func (l *Logger) LogEntry(strMsg string, iMsgLevel int, bAbort bool) {
	strTimeStamp := time.Now().Format("01-02-2006 15:04:05")
	if l.iVerbose > iMsgLevel {
		fmt.Fprintf(l.objLogOut, "%s : %s\n", strTimeStamp, strMsg)
		fmt.Println(strMsg)
	} else if bAbort {
		fmt.Fprintf(l.objLogOut, "%s : %s\n", strTimeStamp, strMsg)
	}
	if bAbort {
		l.Close()
		os.Exit(9)
	}
}

func (l *Logger) Log(strMsg string) {
	l.LogEntry(strMsg, 0, false)
}

func (l *Logger) Close() {
	l.objLogOut.Close()
	fmt.Println("objLogOut closed")
}
