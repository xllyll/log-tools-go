function genMessage(logRuleSet, logExample){
    let msg = `帮我生成一个日志规则，内容: ${logExample}\ 其中:`
    let outRule = []
    if (logRuleSet.time) {
        msg += `${logRuleSet.time}是日志时间,`;
        outRule.push('时间');
    }
    if (logRuleSet.level) {
        msg += `${logRuleSet.level}是日志级别,`;
        outRule.push('日志级别');
    }
    if (logRuleSet.thread) {
        msg += `${logRuleSet.thread}是日志线程,`;
        outRule.push('线程');
    }
    if (logRuleSet.class) {
        msg += `${logRuleSet.class}是日志类名,`;
        outRule.push('类名');
    }
    if (logRuleSet.message) {
        msg += `${logRuleSet.message}是日志内容,`;
        outRule.push('日志内容');
    }
    let r = {
        "timestamp": "(\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2})",
        "timestamp_format": "YYYY-MM-dd hh:mm:ss",
        "level": "(ERROR|INFO|WARN|DEBUG|TRACE|FATAL)",
        "thread": "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2} \\[([^\\]]+)\\]",
        "class": "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2} \\[[^\\]]+\\] \\w+\\s+([a-zA-Z][\\w.]*)",
        "class_line": "",
        "message": "->\\s*(.*)"
    }
    msg += `
            \n借鉴:log: 
            ~~~
            2025-06-27 09:11:06 [main] INFO  c.l.v.dao.redis.RedisDao -> RedisDao init success
            2025-06-27 09:11:06 [main] ERROR c.l.v.dao.redis.RedisDao -> JedisPool is not initialized.
            ~~~
            表达式：${JSON.stringify(r)}
            `
    msg += `要求如下：\n1. 输出${outRule.join(',')}这几个字段对应的正则获取表达式\n`;
    msg += `2. 分开每个需要输出的内容都需要写一个独立的正则匹配表达式，不要合在一起\n`;
    msg += `3. 后面的匹配要加上辅助定位\n`;
    return msg
}