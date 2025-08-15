/**
 * @file app.js
 */
new Vue({
    el: '#app',
    data() {
        return {
            activeTab: 'upload',
            uploadUrl: '/api/upload',
            files: [],
            currentFileId: null,
            logs: [],
            stats: {total_entries: 0, level_counts: {}},
            loading: false,
            currentPage: 0,
            pageSize: 100,
            totalLogs: 0,
            availableLevels: [],
            levelColorMap: {},
            filterForm: {levels: [], keywords: ''},
            selectedProject: '',
            projectList: [],
            showDialog:false,
            settingForm:{
                color:'#666666',
                fontSize: 12,
                threadColor: '#409EFF',
                classColor: '#409EFF',
                showTime: true,
                showThread: true,
                showClass: false,
                showClassLine: false,
            },
            aiShow: false,
            aiLoading: false,
            // aiRes:'根据你提供的日志内容，我们可以从多个角度来分析其中存在的问题。以下是详细的分析和可能的解决方案：\\n\\n---\\n\\n## 🔍 一、总体观察\\n\\n这些日志主要记录了一个车辆通信系统（可能是一个TBox设备）与服务器之间的数据交互过程，包括：\\n\\n- 数据解析（VIN码、经纬度、命令等）\\n- 时间校正逻辑（`terminalCorrectionTime`）\\n- Redis连接状态及异常处理\\n- 网络通信（TCP客户端）信息\\n- 异常日志：如 `JedisPool is not initialized.` 和 `Connection refused`\\n\\n整体来看，系统在正常运行过程中出现了一些**Redis连接失败**的问题，并且存在一些**重复或不一致的数据上报行为**。\\n\\n---\\n\\n## 🧨 二、关键问题分析\\n\\n### 1. **Redis 连接失败 / JedisPool 初始化失败**\\n\\n#### 日志片段：\\n```\\n[Thread-1] ERROR c.l.v.dao.redis.RedisDao -> JedisPool is not initialized.\\n[main] ERROR c.l.v.dao.redis.RedisDao -> Error while closing Jedis is null\\n[main] ERROR c.l.v.dao.redis.RedisDao -> Error while getting Jedis resource\\n...\\nCaused by: redis.clients.jedis.exceptions.JedisConnectionException: Could not get a resource from the pool\\n```\\n\\n#### 问题原因：\\n- **Redis服务未启动** 或 **配置错误**\\n- **JedisPool 没有正确初始化**\\n- **网络不通或防火墙拦截**\\n- **Redis连接池资源耗尽**\\n\\n#### 解决建议：\\n1. **确认 Redis 是否正在运行**：\\n   ```bash\\n   systemctl status redis\\n   # 或者\\n   ps -ef | grep redis\\n   ```\\n\\n2. **检查 Redis 配置文件**（通常是 `/etc/redis/redis.conf`）：\\n   - `bind 127.0.0.1` 应该允许外部访问（如果需要远程连接）\\n   - 端口是否被占用（默认是 6379）\\n\\n3. **代码中确保初始化顺序正确**：\\n   - 在使用前必须先调用 `RedisDao.init()` 方法初始化 JedisPool。\\n   - 避免并发访问未初始化的连接池。\\n\\n4. **增加重试机制或降级策略**：\\n   - 如果 Redis 不可用，应记录日志并跳过缓存操作，避免阻塞主流程。\\n\\n---\\n\\n### 2. **异或校验位数据频繁变化但无明显规律**\\n\\n#### 日志片段：\\n```\\nterminalCorrectionTime, 异或校验位的数据: 79\\nterminalCorrectionTime, 异或校验位的数据: 2d\\nterminalCorrectionTime, 异或校验位的数据: 73\\n...\\n```\\n\\n#### 可能问题：\\n- **校验逻辑存在问题**：校验位应该基于特定字段生成，若每次都变，则说明计算方式不对或者字段不同。\\n- **数据结构未对齐**：比如某些字段缺失或拼接错误导致校验失败。\\n\\n#### 解决建议：\\n- 对比前后几次请求中的原始数据，找出哪些字段参与了异或运算。\\n- 打印完整的原始数据包用于调试校验逻辑。\\n\\n---\\n\\n### 3. **VIN码重复、偏移位置差异小但命令编号递增**\\n\\n#### 日志片段：\\n```\\nvinCodeHax: 4C46335044553043395241303030303530\\nlat: 29.303397000000007, lon: 106.89133100000002\\n>>>cmd:5\\n...\\nvinCodeHax: 4C46335044553043395241303030303530\\nlat: 29.30299700000001, lon: 106.89143100000003\\n>>>cmd:2\\n```\\n\\n#### 可能问题：\\n- 同一 VIN 的位置变动不大（可能是静止状态或GPS漂移）\\n- 命令编号按顺序增长（正常行为），但中间是否有遗漏？\\n- 多个线程同时操作共享变量（例如 cmd 编号），可能导致混乱。\\n\\n#### 解决建议：\\n- 检查是否为多线程并发处理时，命令号分配机制是否安全。\\n- 查看是否有丢失的 cmd 数据，比如从 cmd:2 到 cmd:5 中间是否有跳过。\\n- 若是模拟测试，注意不要让多个线程交叉执行。\\n\\n---\\n\\n### 4. **TCP通信中时间戳与实际接收时间不符**\\n\\n#### 日志片段：\\n```\\nTcpClient, 控制指令: 01 status:01\\nTcpClient, 服务器返回的时间: 2025-06-27 10:59:49\\nTcpClient, 服务器返回的时间: 19061b0a3b31\\n```\\n\\n#### 可能问题：\\n- 时间格式转换异常，比如字符串转时间戳时出错。\\n- 服务器发送的是某种编码后的十六进制时间戳（如 `19061b0a3b31`），需解码后再比较。\\n\\n#### 解决建议：\\n- 根据协议文档判断该字段是否为标准 UTC 时间戳，如果是，请做格式化转换。\\n- 添加调试输出打印出解析后的具体时间值进行比对。\\n\\n---\\n\\n### 5. **Redis 初始化多次触发，且有初始化失败警告**\\n\\n#### 日志片段：\\n```\\nRedis 连接池 初始化成功 Host:127.0.0.1 Port:6379 Password:null\\nJedisPool is not initialized.\\nRedisDao init success\\n```\\n\\n#### 可能问题：\\n- 初始化逻辑重复调用了两次甚至更多。\\n- 存在一个类加载器问题（尤其是在热部署环境下）。\\n- 初始化方法内部存在条件判断失误。\\n\\n#### 解决建议：\\n- 使用单例模式保证只初始化一次。\\n- 加入日志标识符区分初始化来源（如 thread name）。\\n- 审查 `RedisDao.init()` 方法，确保它不是被反复调用。\\n\\n---\\n\\n## ✅ 三、总结建议\\n\\n| 类型 | 描述 | 建议 |\\n|------|------|------|\\n| 💥 Redis 连接问题 | `JedisPool is not initialized`, `Connection refused` | 确保 Redis 正常运行；完善初始化逻辑；添加容错机制 |\\n| 🔐 校验逻辑异常 | 异或校验位频繁变化 | 分析数据源，验证校验算法一致性 |\\n| 📍 地理位置更新 | 位置变动小，命令编号递增 | 检查是否有数据丢失或并发冲突 |\\n| ⏱️ 时间戳解析 | 时间戳格式混乱 | 明确协议定义，统一解析方法 |\\n| 🔄 多线程安全 | 多线程操作 cmd 编号等 | 使用同步机制或原子变量控制 |\\n\\n---\\n\\n如果你能提供以下信息，我可以进一步帮你深入定位问题：\\n\\n1. 相关代码片段（尤其是涉及 Redis 初始化、校验逻辑的部分）\\n2. 协议文档（特别是关于 `vinCodeHax`、`cmd`、`terminalCorrectionTime` 的定义）\\n3. 当前环境配置（Redis 版本、JDK 版本、操作系统）\\n\\n--- \\n\\n需要我继续协助排查某一部分吗？比如帮你写一段校验函数、修复 Redis 初始化流程、优化多线程处理等？',
            aiRes:'',
            aiReq:'',
            logExample:'2025-06-27 09:11:06 [main] INFO  c.l.v.dao.redis.RedisDao -> RedisDao init success',
            logRuleSet:{
                time:'2025-06-27 09:11:06',
                level:'INFO',
                thread:'[main]',
                class:'c.l.v.dao.redis.RedisDao',
                message:'RedisDao init success'
            },
            aiRuleRes:'',
        }
    },
    mounted() {
        // ✅ 正确获取 markedHighlight 插件函数
        const markedHighlightPlugin = window.markedHighlight.default || window.markedHighlight.markedHighlight;
        // 设置 marked 使用 marked-highlight 插件
        marked.use(markedHighlightPlugin({
            langPrefix: 'hljs language-', // highlight.js 的 class 前缀
            highlight: function (code, lang) {
                console.log("lang>",lang)
                const language = hljs.getLanguage(lang) ? lang : 'plaintext';
                return hljs.highlight(code, { language }).value;
            }
        }));
        this.loadFileList();
        this.loadLogLevels();
        this.loadProjects();
    },
    methods: {
        beforeUpload(file) {
            const isValidType = ['.txt', '.log', '.gz', '.zip'].some(ext =>
                file.name.toLowerCase().endsWith(ext)
            );
            if (!isValidType) {
                this.$message.error('只支持 .txt, .log, .gz, .zip 格式的文件');
                return false;
            }
            return true;
        },

        handleUploadSuccess(response, file) {
            if (response.success) {
                this.$message.success(response.message);
                this.loadFileList();
                if (response.file_id) {
                    const fileIds = response.file_id.split(',');
                    this.currentFileId = fileIds[0];
                    this.loadLogs();
                }
            } else {
                this.$message.error(response.error);
            }
        },

        handleUploadError(err) {
            this.$message.error('上传失败: ' + err.message);
        },

        showSettings(){
            console.log('showSettings')
            this.showDialog = true;
        },

        loadFileList() {
            fetch('/api/files')
                .then(response => response.json())
                .then(data => {
                    if (data.success && data.data) {
                        this.files = data.data;
                    }else{
                        this.files = [];
                    }
                })
                .catch(error => {
                    console.error('加载文件列表失败:', error);
                });
        },

        selectFile(fileId) {
            this.currentFileId = fileId;
            this.currentPage = 0;
            this.loadLogs();
        },

        loadLogLevels() {
            fetch('/api/logs/levels')
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        this.availableLevels = data.data;
                        data.data.forEach(item => {
                            this.levelColorMap[item.level] = item.color;
                        })
                    }
                })
                .catch(error => {
                    console.error('加载日志级别失败:', error);
                });
        },

        loadLogs() {
            if (!this.currentFileId) return;

            this.loading = true;
            const params = new URLSearchParams({
                file_id: this.currentFileId,
                limit: this.pageSize,
                offset: this.currentPage * this.pageSize
            });

            if (this.filterForm.levels.length > 0) {
                params.append('levels', this.filterForm.levels.join(','));
            }
            if (this.filterForm.keywords) {
                params.append('keywords', this.filterForm.keywords);
            }

            fetch(`/api/logs?${params}`)
                .then(response => response.json())
                .then(data => {
                    this.loading = false;
                    if (data.success) {
                        if (data.data && data.data.length > 0) {
                            this.logs = data.data;
                            this.stats = data.stats;
                            this.totalLogs = data.stats.total_entries;
                        } else {
                            this.logs = [];
                            this.stats = data.stats;
                            this.totalLogs = 0;
                        }
                    } else {
                        this.$message.error(data.error);
                    }
                })
                .catch(error => {
                    this.loading = false;
                    this.$message.error('加载日志失败: ' + error.message);
                });
        },
        // 新增方法：高亮代码块
        highlightCode() {
            this.$nextTick(() => {
                // hljs.highlightAll()
                document.querySelectorAll('pre code').forEach((block) => {
                    hljs.highlightElement(block);
                });
            });
        },
        buildMarkdownCode(code){
            // code = code.replace(/\\n/g, '\n');
            code = code.replace('\n"', '\\n');
            // 设置 marked 的选项
            let markCode = marked.parse(code, {
                gfm: true,
                breaks: true,
                highlight: function (code, lang) {
                    console.log('>>>>>>>>>>>>>> lang:', lang)
                    if (lang && hljs.getLanguage(lang)) {
                        return hljs.highlight(code, { language: lang }).value;
                    } else {
                        // 自动检测语言（备用）
                        return hljs.highlightAuto(code).value;
                    }
                }
            });
            return markCode;
        },
        async scrollToBottom() {
            await this.$nextTick(); // ✅ 确保 DOM 已更新
            const container = this.$refs.aiContentRef;
            if (container) {
                container.scrollTop = container.scrollHeight;
            }
        },
        analyzeLogs(){
            if (this.logs.length<=0){
                this.$message.error('请选择日志文件！');
                return;
            }
            this.aiShow = true;
            this.aiLoading = true;
            this.highlightCode();
            let logs = [];
            this.logs.forEach(log=>{
                logs.push(log.content);
            })
            const aiMsg = `分析下面的日志文件：\n${logs.join('\n')}`
            let data = {
                logs: aiMsg,
                fileId: this.currentFileId,
            }
            this.startStream(data).catch(err=>{

            })
        },
        async startStream(reqData) {
            const response = await fetch('/api/logs/analysis/stream', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(reqData)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder('utf-8');
            let result = '';

            // 用于通知外部的回调（可选）
            const onText = (data)=> {
                //console.log('Received:', data);
                result += data.msg;
                this.aiRes = result;
                this.scrollToBottom();
            }
            const onDone = ()=>{
                console.log('Done');
                this.aiLoading = false;
            }
            const onError = (error)=>{
                console.log('Error:', error);
                this.$message.error('AI分析失败: ' + error);
            }
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                const chunk = decoder.decode(value, { stream: true });
                const lines = chunk.split('\n');

                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = line.slice(6); // 去掉 "data: "
                        try {
                            const json = JSON.parse(data);
                            if (json.type === 'error') {
                                console.error('AI Error:', json.error);
                                onError(json.error);
                                return;
                            }
                            if (json.type === 'done') {
                                console.log('流结束:', result);
                                onDone();
                                return;
                            }
                            if (json.type === 'stream') {
                                onText(json);
                            }
                        } catch (e) {
                            console.log(">>>>"+data)
                            console.log('解析错误:', e)
                        }
                    }
                }
            }
        },
        applyFilter() {
            this.currentPage = 0;
            this.loadLogs();
        },

        handlePageChange(page) {
            console.log(page)
            this.currentPage = page - 1;
            this.loadLogs();
        },

        handleSizeChange(size){
            console.log(size)
            this.currentPage = 0;
            this.pageSize = size;
            this.loadLogs();
        },

        formatFileSize(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        },

        formatTime(log_time) {
            return new Date(log_time).toLocaleString('zh-CN');
        },

        loadProjects() {
            fetch('/api/projects')
                .then(res => res.json())
                .then(data => {
                    console.log(data);
                    if (data.success) {
                        this.projectList = data.data;
                        this.selectedProject = data.data[0].project_name;
                    }
                });
        },
        handleUpload(option) {
            const file = option.file;
            const formData = new FormData();
            formData.append('file', file);
            formData.append('project_name', this.selectedProject);
            fetch('/api/upload', {
                method: 'POST',
                body: formData
            }).then(res => res.json())
                .then(data => {
                    if (data.success) {
                        this.$message.success('上传成功！');
                        this.loadFileList();
                        // 提示查看
                        this.$alert('上传成功！是否立即查看？', '提示', {
                            confirmButtonText: '立即查看',
                            cancelButtonText: '取消',
                            callback: action => {
                                const fileIds = data.file_id.split(',');
                                this.selectFile(fileIds[0])
                            }
                        });
                    } else {
                        this.$message.error(data.message || '上传失败');
                    }
                }).catch(() => {
                this.$message.error('上传失败');
            });
        },
        removeFile(fileId) {
            fetch(`/api/files/${fileId}`, {
                method: 'DELETE'
            }).then(res => res.json())
                .then(data => {
                    if (data.success) {
                        this.$message.success('删除成功！');
                        this.loadFileList();
                    } else {
                        this.$message.error(data.message || '删除失败');
                    }
                }).catch(() => {
                this.$message.error('删除失败');
            })
        },

        getLevelColor(level){
            let color = '#999999';
            color = this.levelColorMap[level] || color;
            return `4px solid ${color}`;
        },
        highlightKeywords(text) {
            const keywords = [this.filterForm.keywords]
            // 先 HTML 转义，防止 XSS
            const escaped = text
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#x27;');

            // 关键：使用 (A|B|C) 捕获组，而不是 [A|B|C]
            const escapedKeywords = keywords
                .filter(k => k) // 过滤空字符串
                .map(k => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')); // 转义正则特殊字符

            if (escapedKeywords.length === 0) return escaped;

            const regex = new RegExp(`(${escapedKeywords.join('|')})`, 'gi');

            return escaped.replace(regex, '<mark style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: #fff; padding: 0 1px; border-radius: 3px; font-weight: bold;border: 1px solid #ff0000;">$1</mark>');
        },
        aiGenerateLogRule(){
            this.aiLoading = true;
            let msg = genMessage(this.logRuleSet, this.logExample)
            fetch('/api/logs/rule/generate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    log: this.logExample,
                    msg: msg
                })
            }).then(res => res.json()).then(res=>{
                this.aiLoading = false;
                if(res.success){
                    this.aiRuleRes = res.data;
                    this.$message.success('生成成功！');
                }else{
                    this.$message.error(res.message || '生成失败');
                }
            }).catch((err) => {
                this.aiLoading = false;
            })
        },
        resetGenerateLogRule(){
            this.aiRuleRes = '';
        }
    }
});