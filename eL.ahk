﻿#NoEnv  ; Recommended for performance and compatibility with future AutoHotkey releases.
; #Warn  ; Enable warnings to assist with detecting common errors.
SendMode Input  ; Recommended for new scripts due to its superior speed and reliability.
SetWorkingDir %A_ScriptDir%  ; Ensures a consistent starting directory.

Send, {CtrlDown}
Send, {2 Down}
Sleep, 50
Send, {2 Up}
Send, {2 Down}
Sleep, 50
Send, {2 Up}
Send, {CtrlUp}