import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import App from './App.vue'
import './style.css'
const vuetify = createVuetify({ components, directives, theme: { defaultTheme: 'folio', themes: { folio: { dark: false, colors: { primary: '#365848', secondary: '#bf714f', background: '#f7f8fa', surface: '#ffffff' } } } }, defaults: { VBtn: { rounded: 'lg', elevation: 0, style: 'text-transform:none;letter-spacing:0' }, VTextField: { variant: 'outlined', density: 'compact', hideDetails: true }, VSelect: { variant: 'outlined', density: 'compact', hideDetails: true } } })
createApp(App).use(vuetify).mount('#app')
