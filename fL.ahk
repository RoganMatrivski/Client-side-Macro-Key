#NoEnv  ; Recommended for performance and compatibility with future AutoHotkey releases.
; #Warn  ; Enable warnings to assist with detecting common errors.
SendMode Input  ; Recommended for new scripts due to its superior speed and reliability.
SetWorkingDir %A_ScriptDir%  ; Ensures a consistent starting directory.

Send, {CtrlDown}
Send, {3 Down}
Sleep, 50
Send, {3 Up}
Send, {3 Down}
Sleep, 50
Send, {3 Up}
Send, {CtrlUp}