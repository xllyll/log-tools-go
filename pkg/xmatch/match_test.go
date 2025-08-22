package xmatch

import (
	"fmt"
	"log-tools-go/pkg/xlog"
	"regexp"
	"testing"
)

func TestMatch(t *testing.T) {
	str := "2025-06-27 09:11:06 [main] INFO  c.l.v.dao.redis.RedisDao RedisDao init success"
	//pattern := "([a-zA-Z0-9.$]+)\\s*"
	pattern := "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2} \\[[^\\]]+\\] \\w+\\s+([a-zA-Z][\\w.]*)"
	result := Match2(str, pattern)
	println(">>>>>:" + result)
}
func Match2(str string, pattern string) string {
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(str)
	return match[0]
}
func extractClassName(line string) string {
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \[[^\]]+\] \w+\s+([a-zA-Z][\w.]*)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractThreaName(line string) string {
	//p := "^\\{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}.\\d{3}\\s+(\\d)\\s+"
	p := "^(\\{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}.\\d{3})"
	return Match(p, line)
}
func TestCCC(t *testing.T) {
	// ^(\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}\.\d{3})\s+(\d+)\s+(\d+)\s+([A-Z])\s+([^:]+)\s*:\s*(.*)$
	lines := []string{
		`08-02 15:56:24.951  2669  2723 D Sensors : PlsSensor: readEvents (type=3, code=40)`,
		`08-02 15:56:24.951  2669  2723 D Sensors : PlsSensor: mPendingEvents.light = 4913.000000`,
		`08-02 15:56:24.951  2669  2723 D Sensors : PlsSensor: readEvents (type=0, code=0)`,
		`08-02 15:56:24.965  2564  2616 I DeviceService: DispatchThread-DFSK_SD_DVR [write] L0-> aa 44 4d 07 23 00 65`,
		`08-02 15:56:24.966  2564  2616 I VehicleHAL-BW-DFSK-SD-DVR: read persist.vendor.bw.dvr.auto_date:false`,
		`08-02 15:56:24.966  2564  2617 E DeviceService: DeviceService-DFSK_SD_DVR [writeToDevice] :  L0-> aa 44 4d 07 23 00 65`,
		`08-02 15:56:25.032   2564  2601 I DeviceService: DispatchThread-MCU [write] L0-> 55 78 d9 09 50 ea 00 09 07`,
		`08-02 15:56:25.033   2564  2599 E DeviceService: DeviceService-MCU [writeToDevice] :  L0-> 55 78 d9 09 50 ea 00 09 07`,
		`08-02 15:56:25.039  2564    2601 E DeviceService: DispatchThread-MCU [retAck] L0-> 78 55 d9 07 00 4a f0`,
		`08-02 15:56:25.039 2564  2601 E DeviceService: DispatchThread-MCU [notify] L0-> 78 55 c0 0d 00 cc 00 02 00 01 64 02 00`,
		`08-02 15:56:25.039  2564    2600 E VehicleHAL-BW-System: L_ACC:[1,1], C_ACC:[3,3], LCD:[1,1], REVERSE:[0,0]`,
	}

	for _, line := range lines {
		fmt.Printf("\n%s", line)
		pm := `^(\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3})`
		tv := Match(pm, line)
		pm = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}\s+(\d+)`
		pid := Match(pm, line)
		pm = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}\s+\d+\s+(\d+)`
		tid := Match(pm, line)
		pm = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}\s+\d+\s+\d+\s+([A-Z{1}])`
		level := Match(pm, line)
		pm = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}\s+\d+\s+\d+\s+[A-Z{1}]\s+([^:]+)`
		module := Match(pm, line)
		pm = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}\s+\d+\s+\d+\s+[A-Z{1}]\s+[^:]+:\s*([^\s+]+)`
		tag := Match(pm, line)
		pm = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}\s+\d+\s+\d+\s+[A-Z{1}]\s+[^:]+:\s*[^\s+]+\s*(.*)$`
		msg := Match(pm, line)
		fmt.Printf("\n time:%s pid:%s tid:%s level:%s m:%s tag:%s msg:%s", xlog.BlueLog(tv), xlog.CyanLog(pid), xlog.GrayLog(tid), xlog.PurpleLog(level), xlog.GreenLog(module), xlog.RedLog(tag), xlog.YellowLog(msg))
	}
	fmt.Printf("\n")
}
