/**
 * @file app.js
 */
new Vue({
    el: '#app',
    components: {
        'log-view': window.MyApp.Components.LogView,
        'log-header-view': window.MyApp.Components.LogHeaderView,
        'upload-view': window.MyApp.Components.UploadView
    },
    data() {
        return {
            activeTab: 'upload',
            uploadUrl: '/api/upload',
            uploading: false,
            uploadFiles:[],
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
            filterForm: {
                levels: [],
                module: '',
                keywords: '',
                useRegex: false
            },
            projectList: [],
            selectedProject: {},
            selectedProjectName: '',
            selectedModule: {},
            selectedModuleName: '',
            selectedSceneName: '',
            keywords: [],
            showDialog: false,
            settingForm: {
                color: '#666666',
                fontSize: 12,
                threadColor: '#409EFF',
                moduleColor: '#40900F',
                classColor: '#409EFF',
                showAll: true,
                showTime: false,
                showThread: false,
                showModule: false,
                showClass: false,
                showClassLine: false,
            },
            darkMode: false,
            aiShow: false,
            aiLoading: false,
            // aiRes:'根据你提供的日志内容，我们可以从多个角度来分析其中存在的问题。以下是详细的分析和可能的解决方案：\\n\\n---\\n\\n## 🔍 一、总体观察\\n\\n这些日志主要记录了一个车辆通信系统（可能是一个TBox设备）与服务器之间的数据交互过程，包括：\\n\\n- 数据解析（VIN码、经纬度、命令等）\\n- 时间校正逻辑（`terminalCorrectionTime`）\\n- Redis连接状态及异常处理\\n- 网络通信（TCP客户端）信息\\n- 异常日志：如 `JedisPool is not initialized.` 和 `Connection refused`\\n\\n整体来看，系统在正常运行过程中出现了一些**Redis连接失败**的问题，并且存在一些**重复或不一致的数据上报行为**。\\n\\n---\\n\\n## 🧨 二、关键问题分析\\n\\n### 1. **Redis 连接失败 / JedisPool 初始化失败**\\n\\n#### 日志片段：\\n```\\n[Thread-1] ERROR c.l.v.dao.redis.RedisDao -> JedisPool is not initialized.\\n[main] ERROR c.l.v.dao.redis.RedisDao -> Error while closing Jedis is null\\n[main] ERROR c.l.v.dao.redis.RedisDao -> Error while getting Jedis resource\\n...\\nCaused by: redis.clients.jedis.exceptions.JedisConnectionException: Could not get a resource from the pool\\n```\\n\\n#### 问题原因：\\n- **Redis服务未启动** 或 **配置错误**\\n- **JedisPool 没有正确初始化**\\n- **网络不通或防火墙拦截**\\n- **Redis连接池资源耗尽**\\n\\n#### 解决建议：\\n1. **确认 Redis 是否正在运行**：\\n   ```bash\\n   systemctl status redis\\n   # 或者\\n   ps -ef | grep redis\\n   ```\\n\\n2. **检查 Redis 配置文件**（通常是 `/etc/redis/redis.conf`）：\\n   - `bind 127.0.0.1` 应该允许外部访问（如果需要远程连接）\\n   - 端口是否被占用（默认是 6379）\\n\\n3. **代码中确保初始化顺序正确**：\\n   - 在使用前必须先调用 `RedisDao.init()` 方法初始化 JedisPool。\\n   - 避免并发访问未初始化的连接池。\\n\\n4. **增加重试机制或降级策略**：\\n   - 如果 Redis 不可用，应记录日志并跳过缓存操作，避免阻塞主流程。\\n\\n---\\n\\n### 2. **异或校验位数据频繁变化但无明显规律**\\n\\n#### 日志片段：\\n```\\nterminalCorrectionTime, 异或校验位的数据: 79\\nterminalCorrectionTime, 异或校验位的数据: 2d\\nterminalCorrectionTime, 异或校验位的数据: 73\\n...\\n```\\n\\n#### 可能问题：\\n- **校验逻辑存在问题**：校验位应该基于特定字段生成，若每次都变，则说明计算方式不对或者字段不同。\\n- **数据结构未对齐**：比如某些字段缺失或拼接错误导致校验失败。\\n\\n#### 解决建议：\\n- 对比前后几次请求中的原始数据，找出哪些字段参与了异或运算。\\n- 打印完整的原始数据包用于调试校验逻辑。\\n\\n---\\n\\n### 3. **VIN码重复、偏移位置差异小但命令编号递增**\\n\\n#### 日志片段：\\n```\\nvinCodeHax: 4C46335044553043395241303030303530\\nlat: 29.303397000000007, lon: 106.89133100000002\\n>>>cmd:5\\n...\\nvinCodeHax: 4C46335044553043395241303030303530\\nlat: 29.30299700000001, lon: 106.89143100000003\\n>>>cmd:2\\n```\\n\\n#### 可能问题：\\n- 同一 VIN 的位置变动不大（可能是静止状态或GPS漂移）\\n- 命令编号按顺序增长（正常行为），但中间是否有遗漏？\\n- 多个线程同时操作共享变量（例如 cmd 编号），可能导致混乱。\\n\\n#### 解决建议：\\n- 检查是否为多线程并发处理时，命令号分配机制是否安全。\\n- 查看是否有丢失的 cmd 数据，比如从 cmd:2 到 cmd:5 中间是否有跳过。\\n- 若是模拟测试，注意不要让多个线程交叉执行。\\n\\n---\\n\\n### 4. **TCP通信中时间戳与实际接收时间不符**\\n\\n#### 日志片段：\\n```\\nTcpClient, 控制指令: 01 status:01\\nTcpClient, 服务器返回的时间: 2025-06-27 10:59:49\\nTcpClient, 服务器返回的时间: 19061b0a3b31\\n```\\n\\n#### 可能问题：\\n- 时间格式转换异常，比如字符串转时间戳时出错。\\n- 服务器发送的是某种编码后的十六进制时间戳（如 `19061b0a3b31`），需解码后再比较。\\n\\n#### 解决建议：\\n- 根据协议文档判断该字段是否为标准 UTC 时间戳，如果是，请做格式化转换。\\n- 添加调试输出打印出解析后的具体时间值进行比对。\\n\\n---\\n\\n### 5. **Redis 初始化多次触发，且有初始化失败警告**\\n\\n#### 日志片段：\\n```\\nRedis 连接池 初始化成功 Host:127.0.0.1 Port:6379 Password:null\\nJedisPool is not initialized.\\nRedisDao init success\\n```\\n\\n#### 可能问题：\\n- 初始化逻辑重复调用了两次甚至更多。\\n- 存在一个类加载器问题（尤其是在热部署环境下）。\\n- 初始化方法内部存在条件判断失误。\\n\\n#### 解决建议：\\n- 使用单例模式保证只初始化一次。\\n- 加入日志标识符区分初始化来源（如 thread name）。\\n- 审查 `RedisDao.init()` 方法，确保它不是被反复调用。\\n\\n---\\n\\n## ✅ 三、总结建议\\n\\n| 类型 | 描述 | 建议 |\\n|------|------|------|\\n| 💥 Redis 连接问题 | `JedisPool is not initialized`, `Connection refused` | 确保 Redis 正常运行；完善初始化逻辑；添加容错机制 |\\n| 🔐 校验逻辑异常 | 异或校验位频繁变化 | 分析数据源，验证校验算法一致性 |\\n| 📍 地理位置更新 | 位置变动小，命令编号递增 | 检查是否有数据丢失或并发冲突 |\\n| ⏱️ 时间戳解析 | 时间戳格式混乱 | 明确协议定义，统一解析方法 |\\n| 🔄 多线程安全 | 多线程操作 cmd 编号等 | 使用同步机制或原子变量控制 |\\n\\n---\\n\\n如果你能提供以下信息，我可以进一步帮你深入定位问题：\\n\\n1. 相关代码片段（尤其是涉及 Redis 初始化、校验逻辑的部分）\\n2. 协议文档（特别是关于 `vinCodeHax`、`cmd`、`terminalCorrectionTime` 的定义）\\n3. 当前环境配置（Redis 版本、JDK 版本、操作系统）\\n\\n--- \\n\\n需要我继续协助排查某一部分吗？比如帮你写一段校验函数、修复 Redis 初始化流程、优化多线程处理等？',
            aiRes: '',
            aiReq: '',
            logExample: '2025-06-27 09:11:06 [main] INFO  c.l.v.dao.redis.RedisDao -> RedisDao init success',
            logRuleSet: {
                time: '2025-06-27 09:11:06',
                level: 'INFO',
                thread: '[main]',
                class: 'c.l.v.dao.redis.RedisDao',
                message: 'RedisDao init success'
            },
            aiRuleRes: '',
            selectedFileIds: [],
            selectAll: false,
            batchDeleting: false,
            aiMessage:'',
            aiMessages:[
            ]
        }
    },
    mounted() {
        // ✅ 正确获取 markedHighlight 插件函数
        const markedHighlightPlugin = window.markedHighlight.default || window.markedHighlight.markedHighlight;
        // 设置 marked 使用 marked-highlight 插件
        marked.use(markedHighlightPlugin({
            langPrefix: 'hljs language-', // highlight.js 的 class 前缀
            highlight: function (code, lang) {
                console.log("lang>", lang)
                const language = hljs.getLanguage(lang) ? lang : 'plaintext';
                return hljs.highlight(code, {language}).value;
            }
        }));
        this.initDarkMode();
        this.loadFileList();
        this.loadLogLevels();
        this.loadProjects();
    },
    methods: {
        handleUploadSuccess(file) {
            console.log('handleUploadSuccess', file)
            this.$alert('上传成功！是否立即查看？', '提示', {
                confirmButtonText: '立即查看',
                cancelButtonText: '取消',
                callback: action => {
                    this.selectedFileIds = [file.fileId];
                    this.loadLogs();
                    this.loadFileList();
                }
            });
        },
        showSettings() {
            console.log('showSettings')
            this.showDialog = true;
        },
        onSelectProject(e) {
            console.log('onSelectProject', e)
            const ls = this.projectList
            for (let i = 0; i < ls.length; i++) {
                if (ls[i].project_name === e) {
                    this.selectedProject = ls[i];
                    break;
                }
            }
        },
        onSelectModule(e) {
            console.log('onSelectModule', e)
            if (e == null || e === '') {
                this.selectedModule = null;
                this.selectedSceneName = null;
                this.keywords = [];
                this.filterForm.module = null;
                return
            }
            console.log('onSelectModule', e.toString())
            this.selectedProject.modules.forEach(module => {
                if (module.name === e.toString()) {
                    this.selectedModule = module;
                }
            })
            console.log('selectedModule end:', this.selectedModule)
        },
        onSelectModuleScene(e) {
            console.log('onSelectModuleScene', e)
            if (e == null || e === '') {
                this.selectedSceneName = null;
                this.keywords = [];
            }
            if (this.selectedModule) {
                this.selectedModule.scenes.forEach(scene => {
                    if (scene.name === e.toString()) {
                        this.keywords = scene.keywords;
                    }
                })
            }
            console.log('keywords', this.keywords)
        },
        loadFileList() {
            fetch('/api/files')
                .then(response => response.json())
                .then(data => {
                    console.log('loadFileList', data)
                    if (data.success && data.data) {
                        this.files = data.data;
                    } else {
                        this.files = [];
                    }
                })
                .catch(error => {
                    this.files = [];
                    console.error('加载文件列表失败:', error);
                });
        },
        initDarkMode() {
            const savedMode = localStorage.getItem('darkMode');
            if (savedMode === 'true') {
                this.darkMode = true;
                this.toggleDarkMode();
            }
        },

        selectFile(fileId) {
            this.currentFileId = fileId;
            this.currentPage = 0;
            this.loadLogs();
        },
        // 新增切换文件选择状态的方法
        toggleFileSelection(fileId) {
            // 切换文件的选中状态
            const index = this.selectedFileIds.indexOf(fileId);
            if (index > -1) {
                // 如果已选中，则取消选中
                this.selectedFileIds.splice(index, 1);
            } else {
                // 如果未选中，则添加到选中列表
                this.selectedFileIds.push(fileId);
            }

            // 更新全选状态
            this.selectAll = this.selectedFileIds.length === this.files.length && this.files.length > 0;

            // 如果选择了多个文件，自动触发查询
            if (this.selectedFileIds.length > 1) {
                this.loadLogs();
            }
            // 如果只选择了一个文件，则查询该文件
            else if (this.selectedFileIds.length === 1) {
                this.selectFile(this.selectedFileIds[0]);
            }
            // 如果没有选择文件，则清空日志显示
            else {
                this.logs = [];
                this.totalLogs = 0;
            }
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
        /** 加载日志内容列表 */
        loadLogs() {
            if (this.selectedFileIds.length === 0) {
                this.logs = [];
                this.totalLogs = 0;
                this.stats = {};
                return;
            }
            this.loading = true;

            // 构造请求数据
            const requestData = {
                file_ids: this.selectedFileIds.join(','),
                limit: this.pageSize,
                offset: this.currentPage * this.pageSize
            };

            // 添加过滤条件
            if (this.filterForm.module && this.filterForm.module !== '') {
                requestData.module = this.filterForm.module;
            }
            if (this.filterForm.levels.length > 0) {
                requestData.levels = this.filterForm.levels;
            }
            if (this.filterForm.keywords) {
                requestData.keywords = this.filterForm.keywords.split(',');
            }

            // 使用POST请求替代原来的GET请求
            fetch('/api/logs', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            })
                .then(response => response.json())
                .then(data => {
                    debugger
                    this.loading = false;
                    if (data.success) {
                        this.stats = data.stats;
                        this.totalLogs = data.stats.total_entries || 0;
                        if (data.data && data.data.length > 0) {
                            this.logs = data.data;
                        } else {
                            this.logs = [];
                        }
                    } else {
                        this.logs = [];
                        this.totalLogs = 0;
                        this.$message.error(data.error || '获取日志失败');
                    }
                })
                .catch(error => {
                    this.loading = false;
                    console.error('获取日志失败:', error);
                    this.$message.error('获取日志失败: ' + error.message);
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
        buildMarkdownCode(code) {
            // code = code.replace(/\\n/g, '\n');
            code = code.replace('\n"', '\\n');
            // 设置 marked 的选项
            let markCode = marked.parse(code, {
                gfm: true,
                breaks: true,
                highlight: function (code, lang) {
                    console.log('>>>>>>>>>>>>>> lang:', lang)
                    if (lang && hljs.getLanguage(lang)) {
                        return hljs.highlight(code, {language: lang}).value;
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
        analyzeLogs() {
            if (this.logs.length <= 0) {
                this.$message.error('请选择日志文件！');
                return;
            }
            this.aiShow = true;
            this.highlightCode();
            let logs = [];
            this.logs.forEach(log => {
                logs.push(log.content);
            })
            const aiMsg = `分析下面的日志文件：\n${logs.join('\n')}`
            this.startAiChatStream(aiMsg).catch(err => {

            })
        },
        async startAiChatStream(msg) {
            let msgs = this.aiMessages;
            msgs.push({
                role: 'user',
                content: msg,
            });
            const playload = {
                module: 'qwen3-max',
                messages: msgs
            }
            this.aiLoading = true;
            const response = await fetch('/api/ai/chat/completions', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(playload),
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder('utf-8');
            let result = '';
            this.aiRes = "";
            // 用于通知外部的回调（可选）
            const onText = (data) => {
                //console.log('Received:', data);
                result += data.msg;
                this.aiRes = result;
                this.scrollToBottom();
            }
            const onDone = () => {
                console.log('Done');
                this.aiLoading = false;
                this.aiMessages.push({
                    role: 'assistant',
                    content: this.aiRes,
                })
                this.aiRes = "";
            }
            const onError = (error) => {
                console.log('Error:', error);
                this.$message.error('AI分析失败: ' + error);
            }
            while (true) {
                const {done, value} = await reader.read();
                if (done) break;

                const chunk = decoder.decode(value, {stream: true});
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
                            console.log(">>>>" + data)
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

        handleSizeChange(size) {
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
        toggleDarkMode() {
            this.darkMode = !this.darkMode;
            if (this.darkMode) {
                document.body.classList.add('dark-theme');
                const link = document.createElement('link');
                link.rel = 'stylesheet';
                link.href = '/static/css/dark-theme.css';
                link.id = 'dark-theme-style';
                document.head.appendChild(link);
            } else {
                document.body.classList.remove('dark-theme');
                const link = document.getElementById('dark-theme-style');
                if (link) {
                    link.remove();
                }
            }
            // 保存用户偏好
            localStorage.setItem('darkMode', this.darkMode);
        },
        loadProjects() {
            baseRequest('/api/projects', 'GET', null, {}).then(data=>{
                if (data.success) {
                    this.projectList = data.data;
                    if (data.data.length > 0) {
                        this.selectedProject = data.data[0]
                        this.selectedProjectName = data.data[0].project_name;
                    }
                }else{
                    this.projectList = [];
                }
            }).catch( error=>{
                console.error('加载项目列表失败:', error);
            })
        },
        removeFile(fileId) {
            this.$confirm('确定要删除这个文件吗?', '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }).then(() => {
                this.batchDeleting = true;
                baseRequest(`/api/files/${fileId}`, 'DELETE', null, {}).then(data => {
                    this.batchDeleting = false;
                    if (data.success) {
                        this.$message.success('删除成功！');
                        if (this.currentFileId && this.currentFileId === fileId){
                            this.logs = [];
                            this.totalLogs = 0;
                        }
                        // 删除this.selectedFileIds 的文件id
                        this.selectedFileIds = this.selectedFileIds.filter(id => id !== fileId);
                        this.loadFileList();
                    } else {
                        this.$message.error(data.message || '删除失败');
                    }
                }).catch(() => {
                    this.$message.error('删除失败');
                })
            }).catch(() => {
                this.batchDeleting = false;
                this.$message.info('已取消删除');
            });
        },
        // 批量删除文件
        batchRemoveFiles() {
            if (this.selectedFileIds.length === 0) {
                this.$message.warning('请至少选择一个文件');
                return;
            }
            this.$confirm(`确定要删除选中的 ${this.selectedFileIds.length} 个文件吗?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }).then(() => {
                this.batchDeleting = true;
                baseRequest('/api/files/batch-delete', 'POST', {ids: this.selectedFileIds},{'Content-Type': 'application/json'}).then(data=>{
                    this.batchDeleting = false;
                    if (data.success) {
                        this.$message.success(data.message);
                        this.selectedFileIds = [];
                        this.selectAll = false;
                        this.logs = [];
                        this.totalLogs = 0;
                        this.loadFileList();
                    } else {
                        this.$message.error(data.message || '删除失败');
                    }
                }).catch(() => {
                    this.batchDeleting = false;
                    this.$message.error('删除失败');
                })
            }).catch(() => {
                this.$message.info('已取消删除');
            });
        },
        // 处理文件选择变化
        handleFileSelectChange(selected, fileId) {
            if (selected) {
                // 添加到选中列表
                if (!this.selectedFileIds.includes(fileId)) {
                    this.selectedFileIds.push(fileId);
                }
            } else {
                // 从选中列表移除
                this.selectedFileIds = this.selectedFileIds.filter(id => id !== fileId);
            }

            // 更新全选状态
            this.selectAll = this.selectedFileIds.length === this.files.length && this.files.length > 0;

            // 如果选择了多个文件，自动触发查询
            if (this.selectedFileIds.length > 1) {
                this.loadLogs();
            }
        },
        // 处理全选变化
        handleSelectAllChange(selectAll) {
            if (selectAll) {
                // 全选
                this.selectedFileIds = this.files.map(file => file.id);
            } else {
                // 取消全选
                this.selectedFileIds = [];
            }

            // 如果选择了多个文件，自动触发查询
            if (this.selectedFileIds.length > 1) {
                this.loadLogs();
            }
        },
        getLevelColor(level) {
            let color = '#999999';
            switch (level) {
                case 'D':
                    level = 'DEBUG';
                    break
                case 'I':
                    level = 'INFO';
                    break
                case 'W':
                    level = 'WARN';
                    break
                case 'E':
                    level = 'ERROR';
                    break
                case 'F':
                    level = 'FATAL';
                    break
                default:
                    level = 'UNKNOWN';
            }
            color = this.levelColorMap[level] || color;
            return `4px solid ${color}`;
        },
        highlightKeywords(text) {
            let line = this.formatSceneKeywords(text);
            if (line !== null) {
                return line;
            }
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
        formatSceneKeywords(line) {
            if (this.keywords != null) {
                for (let i = 0; i < this.keywords.length; i++) {
                    let key = this.keywords[i];
                    let keyword = key.keyword;
                    if (line.includes(keyword)) {
                        line = `
                            <div class="tooltip-view">
                              <div class="tooltip-view-content">${key.desc}</div>
                              <div class="scene_line">
                                <span class="v1" style="color: ${key.color || '#667eea'}">${line}</span>
                                <span class="v2">${key.desc}</span>
                              </div>
                            </div>
                        `
                        return line
                    }
                }
            }
            return null;
        },
        aiGenerateLogRule() {
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
            }).then(res => res.json()).then(res => {
                this.aiLoading = false;
                if (res.success) {
                    this.aiRuleRes = res.data;
                    this.$message.success('生成成功！');
                } else {
                    this.$message.error(res.message || '生成失败');
                }
            }).catch((err) => {
                this.aiLoading = false;
            })
        },
        resetGenerateLogRule() {
            this.aiRuleRes = '';
        },
        /**
         * 发送消息[继续追问]
         */
        handleAiMessageSend(){
            this.startAiChatStream(this.aiMessage);
        }
    }
});
