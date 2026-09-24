import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import './style.css'
import './styles/theme.css'
// ElMessageBox/ElMessage 是函数式 API，unplugin 按需解析器不覆盖其样式——必须手动引入
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/message/style/css'

createApp(App).use(createPinia()).use(router).mount('#app')
