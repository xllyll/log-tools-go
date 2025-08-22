// 定义一个按钮组件
const LogView = {
    props: ['color', 'label'],
    data() {
        return {
            clickCount: 0
        };
    },
    template: `
    <div>
        <el-tag>123</el-tag>
        <button 
          :style="{ backgroundColor: color, padding: '10px', margin: '5px', border: 'none', color: 'white', cursor: 'pointer' }"
          @click="handleClick">
          {{ label }} (点击了 {{ clickCount }} 次)
        </button>
    </div>
  `,
    methods: {
        handleClick() {
            this.clickCount++;
            // 触发事件，供父组件监听
            this.$emit('click', this.clickCount);
        }
    }
};
