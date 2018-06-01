<template>
    <div ref="contextmenu" class="contextmenu-content" :style="axisComputed" v-if="show" contextmenu='1'>
        <a v-for="item in contextMenuData.menulists" @click.stop="menuHandler(item)">{{item.caption}}</a>
    </div>
</template>

<script>
    export default {
        data() {
            return {
                show: false,
                axis: {x: 0, y: 0}
            }
        },
        props: {
            contextMenuData: {
                type: Object,
                requred: true
            },
            tag: {
                type: String,
                requred: false
            }
        },
        mounted() {
            this.$root.$on('showVueContextMenu', (args) => {
                if (args.tag == this.tag) {
                    this.show = true
                    this.axis = {x: args.x, y: args.y};
                }
            });

            document.addEventListener('mousedown', (e) => {
                if (e.target.tagName == 'A')
                    return;
                this.show = false;
            }, true);
        },
        updated() {
            if (this.$refs.contextmenu) {
                let bw = document.body.offsetWidth;
                let bh = document.body.offsetHeight;
                let width = this.$refs.contextmenu.offsetWidth;
                let height = this.$refs.contextmenu.offsetHeight;

                if (this.axis.x + width >= bw)
                    this.axis.x = bw - width;

                if (this.axis.y + height >= bh)
                    this.axis.y = bh - height;
            }
        },
        computed: {
            axisComputed() {
                return {
                    top: this.axis.y + 'px',
                    left: this.axis.x + 'px'
                }
            }
        },
        methods: {
            menuHandler (item) {
                this.show = false;
                this.$emit('handleContextMenu', item.name);
            }
        }
    }
</script>

<style scoped>
    .contextmenu-content {
        position: fixed;
        z-index: 9999;
        background-color: #f9f9f9;
        min-width: 160px;
        box-shadow: 0px 8px 16px 0px rgba(0, 0, 0, 0.2);
    }

    .contextmenu-content a {
        color: black;
        padding: 5px 16px;
        text-decoration: none;
        display: block;
    }

    .contextmenu-content a:hover {
        background-color: #90C8F6;
    }
</style>
