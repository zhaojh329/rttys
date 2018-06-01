import VueContextMenuComponent from './VueContextMenu.vue'

const VueContextMenu  = {
	install: function (Vue) {
		Vue.component('VueContextMenu', VueContextMenuComponent)

		Vue.prototype.$vuecontextmenu = function (e, root, id) {
			e.stopPropagation();
			e.preventDefault();

			root.$emit('showVueContextMenu', {
				id: id,
				x: e.clientX,
				y: e.clientY
			})
		}
	}
}

if (typeof window !== 'undefined' && window.Vue) {
    window.Vue.use(VueContextMenu)
}

export default VueContextMenu
