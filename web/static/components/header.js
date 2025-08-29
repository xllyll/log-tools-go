const LogHeaderView = {
    name: 'LogHeaderView',
    template: `
        <div class="header">
            <h2 style="margin: 20px;">
                <i class="el-icon-document"></i>
                <span>蓝鲸智联日志分析工具</span>
            </h2>
            <div style="display: flex; align-items: center;">
                <el-tooltip :content="darkMode ? '切换到亮色模式' : '切换到暗黑模式'" placement="bottom">
                    <el-button 
                        :icon="darkMode ? 'el-icon-sunny' : 'el-icon-moon'" 
                        circle 
                        @click="toggleDarkMode"
                        style="margin-right: 10px;"
                    ></el-button>
                </el-tooltip>
                <el-button icon="el-icon-setting" circle @click="handleClick" style="margin-right: 20px;"></el-button>
            </div>
        </div>
    `,
    data() {
        return {
            darkMode: false
        };
    },
    methods: {
        handleClick() {
            console.log('==================>')
            // 触发事件，供父组件监听
            this.$emit('on-setting');
        },
        toggleDarkMode() {
            this.darkMode = !this.darkMode;
            this.$emit('dark-mode-toggle', this.darkMode);
        }
    }
}
