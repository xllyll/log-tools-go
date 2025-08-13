package xmatch

import (
	"fmt"
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
	p := "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2} \\[([^\\]]+)\\]"
	return Match(line, p)
}
func TestCCC(t *testing.T) {
	lines := []string{
		`2025-06-27 09:11:06 [main] INFO  c.l.v.dao.redis.RedisDao RedisDao init success`,
		`2025-06-27 09:11:07 [Thread-1] INFO  c.l.v.dao.redis.RedisDao2 RedisDao2 init success`,
		`2025-06-27 09:11:08 [test] DEBUG com.example.ServiceV2 Starting service...`,
		`2025-06-27 09:11:12 [main] DEBUG com.example.ServiceV2 Starting service...`,
		`redis.clients.jedis.exceptions.JedisConnectionException: Could not get a resource from the pool`,
		`2025-06-27 09:11:13 [main] ERROR com.example.ServiceV2 Error occurred while starting service`,
		`2025-06-27 10:59:49 [Thread-1] INFO  c.l.v.socket.SocketTest -> >>>cmd:0`,
	}

	for _, line := range lines {
		v := extractThreaName(line)
		fmt.Println("提取>>:", v)
	}
}
