import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { initTheme } from './utils/theme'
import './styles/theme.css'
import App from './App.vue'

initTheme()
createApp(App).use(ElementPlus, { locale: zhCn }).mount('#app')
