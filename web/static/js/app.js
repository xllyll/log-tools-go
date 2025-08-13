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
            }
        }
    },
    mounted() {
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
                        if (data.data.length > 0) {
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

        applyFilter() {
            this.currentPage = 0;
            this.loadLogs();
        },

        handlePageChange(page) {
            this.currentPage = page - 1;
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
                        this.selectedProject = data.data[0];
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
            console.log(level,color)
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
        }
    }
})
;