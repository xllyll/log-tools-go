package xlog

import "fmt"

const (
	_reset  = "\033[0m"
	_red    = "\033[31m"
	_green  = "\033[32m"
	_yellow = "\033[33m"
	_blue   = "\033[34m"
	_purple = "\033[35m"
	_cyan   = "\033[36m"
	_gray   = "\033[37m"
	_white  = "\033[97m"
)

func Log(msg string, color string) string {
	return fmt.Sprintf("%s%s%s", color, msg, color)
}
func RestLog(msg string) string {
	return Log(msg, _reset)
}
func RedLog(msg string) string {
	return Log(msg, _red)
}
func GreenLog(msg string) string {
	return Log(msg, _green)
}
func YellowLog(msg string) string {
	return Log(msg, _yellow)
}
func BlueLog(msg string) string {
	return Log(msg, _blue)
}
func PurpleLog(msg string) string {
	return Log(msg, _purple)
}
func CyanLog(msg string) string {
	return Log(msg, _cyan)
}
func GrayLog(msg string) string {
	return Log(msg, _gray)
}
func WhiteLog(msg string) string {
	return Log(msg, _white)
}
