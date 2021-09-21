import Vue from 'vue';
import 'view-design/dist/styles/iview.css'
import en from 'view-design/dist/locale/en-US'
import zh from 'view-design/dist/locale/zh-CN'
import {
    locale, Icon, Message, Card, Modal, Form, FormItem, Input, Button,
    Table, Tooltip, Select, Option, Dropdown, DropdownMenu, DropdownItem,
    Progress, Tag, Upload
} from 'view-design'

if (navigator.language === 'zh-CN')
    locale(zh)
else
    locale(en)

Vue.prototype.$Message = Message
Vue.prototype.$Modal = Modal

Vue.component('Icon', Icon)
Vue.component('Card', Card)
Vue.component('Modal', Modal)
Vue.component('Form', Form)
Vue.component('FormItem', FormItem)
Vue.component('Input', Input)
Vue.component('Button', Button)
Vue.component('Table', Table)
Vue.component('Tooltip', Tooltip)
Vue.component('Select', Select)
Vue.component('Option', Option)
Vue.component('Dropdown', Dropdown)
Vue.component('DropdownMenu', DropdownMenu)
Vue.component('DropdownItem', DropdownItem)
Vue.component('Progress', Progress)
Vue.component('Tag', Tag)
Vue.component('Upload', Upload)