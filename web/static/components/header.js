const LogHeaderView = {
    name: 'LogHeaderView',
    template: `
        <div class="header">
            <h2 style="margin: 20px;">
                <i class="el-icon-document"></i>
                <span>蓝鲸智联日志分析工具</span>
            </h2>
            <el-button icon="el-icon-setting" circle @click="handleClick" style="margin-right: 20px;"></el-button>
        </div>
    `,
    data() {
        return {

        };
    },
    methods: {
        handleClick() {
            console.log('==================>')
            // 触发事件，供父组件监听
            this.$emit('on-setting');
        }
    }
}
